package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LuoZero-World/distributed-lock/internal/utils"
	"github.com/redis/go-redis/v9"
)

const REDIS_LOCK_KEY_PREFIX = "redis:lock:"

var (
	ErrLockAcquiredByOthers   = errors.New("lock is acquired by others")
	ErrUnlockWithoutOwnership = errors.New("can not unlock without ownership of lock")
)

type RedisLock struct {
	LockOptions
	key    string
	token  string // 表示分布式环境下[主机_进程_协程]想要获得此锁
	client *redis.Client
}

type LockOptions struct {
	//是否阻塞 以及阻塞等待时间
	isBlock             bool
	blockWaitingSeconds int64
	//过期时间
	expireSeconds int64
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

// 尝试加锁
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
			return nil
		} else if !IsRetryableErr(err) {
			return err
		}
	}

	//不可达
	return nil
}

func IsRetryableErr(err error) bool {
	return errors.Is(err, ErrLockAcquiredByOthers)
}
