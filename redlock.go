package lock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrTooFewNodes                    = errors.New("can not use redLock less than 3 nodes")
	ErrTooLongExpireTimeForSingleNode = errors.New("expire thresholds of single node is too long")
	ErrTooFewLockSuccessed            = errors.New("lock failed")
)

type RedLock struct {
	locks []*RedisLock
	RedLockOptions
}

type RedLockOptions struct {
	singleNodesTimeout time.Duration //单个锁节点加锁操作的超时时间
	expireDuration     time.Duration //整个红锁中所有锁节点的过期时间
}

func NewRedLock(key string, clients []*redis.Client, opts ...RedLockOption) (*RedLock, error) {
	// 3 个节点及以上，红锁才有意义
	if len(clients) < 3 {
		return nil, ErrTooFewNodes
	}

	r := &RedLock{}
	for _, opt := range opts {
		opt(&r.RedLockOptions)
	}
	repairRedLock(&r.RedLockOptions)
	if time.Duration(len(clients))*r.singleNodesTimeout*10 > r.expireDuration {
		return nil, ErrTooLongExpireTimeForSingleNode
	}

	r.locks = make([]*RedisLock, 0, len(clients))
	for _, client := range clients {
		r.locks = append(r.locks, NewRedisLock(key, client, WithExpireSeconds(int64(r.expireDuration.Seconds()))))
	}

	return r, nil
}

func (r *RedLock) Lock(ctx context.Context) error {
	var successCnt int
	for _, lock := range r.locks {
		startTime := time.Now()
		sonctx, cancel := context.WithTimeout(ctx, r.singleNodesTimeout)
		err := lock.Lock(sonctx)
		cost := time.Since(startTime)
		if err == nil && cost <= r.singleNodesTimeout {
			successCnt++
		}
		cancel() //显示释放资源，而不是 defer cancel()
	}

	if successCnt < len(r.locks)>>1+1 {
		//加锁失败 解锁
		r.Unlock(ctx)
		return ErrTooFewLockSuccessed
	}
	return nil
}

func (r *RedLock) Unlock(ctx context.Context) {
	for _, lock := range r.locks {
		_ = lock.Unlock(ctx)
	}
}
