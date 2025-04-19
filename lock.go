package lock

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/LuoZero-World/distributed-lock/internal/utils"
	"github.com/redis/go-redis/v9"
)

const REDIS_LOCK_KEY_PREFIX = "redis:lock:"

var (
	ErrLockAcquiredByOthers   = errors.New("lock is acquired by others")
	ErrUnlockWithoutOwnership = errors.New("can not unlock without ownership of lock")
	ErrExpireWithoutOwnership = errors.New("can not expire lock without ownership of lock")
)

type RedisLock struct {
	LockOptions
	key    string
	token  string // 表示分布式环境下[主机_进程_协程]想要获得此锁
	client *redis.Client

	//看门狗运行标识 0未运行 1正在运行
	runningWatchDog int32
	//停止看门狗
	stopWatchDog context.CancelFunc
}

type LockOptions struct {
	//是否阻塞 以及阻塞等待时间
	isBlock             bool
	blockWaitingSeconds int64
	//过期时间
	expireSeconds int64
	//是否启用看门狗模式
	watchDogMode bool
}

func NewRedisLock(key string, client *redis.Client, opts ...LockOption) *RedisLock {
	rLock := RedisLock{
		key:    REDIS_LOCK_KEY_PREFIX + key,
		token:  utils.GenerateID(),
		client: client,
	}
	for _, opt := range opts {
		opt(&rLock.LockOptions)
	}
	repairLock(&rLock.LockOptions)
	return &rLock
}

func (rLock *RedisLock) Lock(ctx context.Context) (err error) {

	defer func() {
		//加锁成功，尝试启用看门狗
		if err == nil {
			rLock.watchDog(ctx)
		}
	}()

	// 无论是否阻塞都先尝试加一次锁
	err = rLock.tryLock(ctx)
	if err == nil { //加锁成功
		return nil
	}
	if !rLock.isBlock { //非阻塞情况直接返回
		return
	}
	// 阻塞情况
	if !IsRetryableErr(err) {
		return
	}
	err = rLock.blockingLock(ctx)
	return
}

func (rLock *RedisLock) Unlock(ctx context.Context) (err error) {
	defer func() {
		if err == nil && rLock.stopWatchDog != nil {
			fmt.Println("stop watch dog")
			rLock.stopWatchDog()
		}
	}()

	val, err := rLock.client.Eval(ctx, LuaCheckAndDeleteDistributionLock, []string{rLock.key}, []interface{}{rLock.token}).Result()
	if err != nil {
		return err
	}

	if ret, _ := val.(int64); ret != 1 {
		// Lua脚本返回值不为1 解锁失败表示 无所有权
		return ErrUnlockWithoutOwnership
	}
	return nil
}

// 尝试加锁 SetNX表示了该锁是不可重入的
func (rLock *RedisLock) tryLock(ctx context.Context) error {
	ok, err := rLock.client.SetNX(ctx, rLock.key, rLock.token, time.Duration(rLock.expireSeconds)*time.Second).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s", ErrLockAcquiredByOthers, rLock.key)
	}
	return nil
}

func (rLock *RedisLock) blockingLock(ctx context.Context) error {
	//阻塞模式等待时间上限
	timeoutCh := time.After(time.Duration(rLock.blockWaitingSeconds) * time.Second)
	ticker := time.NewTicker(time.Duration(50) * time.Millisecond)

	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ctx.Done():
			return fmt.Errorf("lock failed, ctx timeout err: %w", ctx.Err())
		case <-timeoutCh:
			return fmt.Errorf("block waiting time out, err: %w", ErrLockAcquiredByOthers)
		default:
		}

		//尝试取锁
		if err := rLock.tryLock(ctx); err == nil { //取锁成功
			fmt.Println("lock with block success")
			return nil
		} else if !IsRetryableErr(err) {
			return err
		}
	}

	//不可达
	return nil
}

func (rLock *RedisLock) watchDog(ctx context.Context) {
	if !rLock.watchDogMode {
		return
	}

	//自旋锁确保只有一个看门狗协程在运行，针对的群体是使用当前rLock的goruntine
	for !atomic.CompareAndSwapInt32(&rLock.runningWatchDog, 0, 1) {
		time.Sleep(10 * time.Millisecond)
	}

	//启动看门狗
	ctx, rLock.stopWatchDog = context.WithCancel(ctx)
	go func() {
		defer atomic.StoreInt32(&rLock.runningWatchDog, 0)
		rLock.runWatchDog(ctx)
	}()
}

func (rLock *RedisLock) runWatchDog(ctx context.Context) {
	watchDogWorkStepSeconds := min(DefaultWatchDogWorkStepSeconds, rLock.expireSeconds)
	ticker := time.NewTicker(time.Duration(watchDogWorkStepSeconds) * time.Second)

	for range ticker.C {
		select {
		case <-ctx.Done(): //当启用看门狗的主goruntine奔溃时，看门狗会从这里退出
			return
		default:
		}

		//用户未显式解锁，自动续期
		_ = rLock.DelayExpire(ctx, watchDogWorkStepSeconds+5)
	}
}

func (rLock *RedisLock) DelayExpire(ctx context.Context, expireSeconds int64) error {
	val, err := rLock.client.Eval(ctx, LuaCheckAndExpireDistributionLock, []string{rLock.key}, []interface{}{rLock.token, expireSeconds}).Result()
	if err != nil {
		return err
	}

	if ret, _ := val.(int64); ret != 1 {
		return ErrExpireWithoutOwnership
	}

	fmt.Println("expire success!")
	return nil
}

func IsRetryableErr(err error) bool {
	return errors.Is(err, ErrLockAcquiredByOthers)
}
