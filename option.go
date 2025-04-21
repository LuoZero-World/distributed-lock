package lock

import (
	"time"
)

const (
	// 默认的阻塞等待时间 s
	DefaultBlockSeconds = 5
	// 默认的分布式锁过期时间 s
	DefaultLockExpireSeconds = 30

	//看门狗工作时间步 s
	DefaultWatchDogWorkStepSeconds = 15

	//红锁中每个节点的默认加锁操作 超时时间
	DefaultSingleNodeTimeout = 50 * time.Millisecond
	//整个红锁中所有锁节点的默认过期时间
	DefaultRedLockExpireDuration = 5 * time.Second
)

type LockOption func(*LockOptions)
type RedLockOption func(*RedLockOptions)

// -------RedisLock-------
func WithBlock() LockOption {
	return func(lo *LockOptions) {
		lo.isBlock = true
	}
}

func WithBlockWaitingSecond(blockSeconds int64) LockOption {
	return func(lo *LockOptions) {
		lo.blockWaitingSeconds = blockSeconds
	}
}

func WithExpireSeconds(expireSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.expireSeconds = expireSeconds
	}
}

func WithWatchDogMode() LockOption {
	return func(o *LockOptions) {
		o.watchDogMode = true
	}
}

func repairLock(opts *LockOptions) {
	if opts.isBlock && opts.blockWaitingSeconds <= 0 {
		opts.blockWaitingSeconds = DefaultBlockSeconds
	}
	if opts.expireSeconds <= 0 {
		opts.expireSeconds = DefaultLockExpireSeconds
		opts.watchDogMode = true //用户未显式指定锁的过期时间，则此时会启动看门狗
	}

}

// -------RedLock-------
func WithSingleNodeTimeout(timeout time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.singleNodesTimeout = timeout
	}
}

func WithRedLockExpireDuration(expireDuration time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.expireDuration = expireDuration
	}
}

func repairRedLock(o *RedLockOptions) {
	if o.singleNodesTimeout <= 0 {
		o.singleNodesTimeout = DefaultSingleNodeTimeout
	}
	if o.expireDuration <= 0 {
		o.expireDuration = DefaultRedLockExpireDuration
	}
}
