package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/LuoZero-World/distributed-lock/utils"
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

func (rLock *RedisLock) Lock(ctx context.Context) error {
	// 无论是否阻塞都先尝试加一次锁
	ok, err := rLock.tryLock(ctx)
	if ok {
		return nil
	}
	if !rLock.isBlock { //非阻塞情况直接返回
		return err
	}
	// TODO 阻塞情况
	return nil
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
func (rLock *RedisLock) tryLock(ctx context.Context) (bool, error) {
	ok, err := rLock.client.SetNX(ctx, rLock.key, rLock.token, time.Duration(rLock.expireSeconds)*time.Second).Result()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrLockAcquiredByOthers, rLock.key)
	}
	return true, nil
}
