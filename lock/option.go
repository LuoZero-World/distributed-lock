package lock

const (
	// 默认的阻塞等待时间
	DefaultBlockSeconds = 5
	// 默认的分布式锁过期时间
	DefaultLockExpireSeconds = 30
)

type LockOption func(*LockOptions)

func WithBlock() LockOption {
	return func(lo *LockOptions) {
		lo.isBlock = true
	}
}

func WithBlockWaitingSecond(expireSeconds int64) LockOption {
	return func(lo *LockOptions) {
		lo.expireSeconds = expireSeconds
	}
}

func WithExpireSeconds(expireSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.expireSeconds = expireSeconds
	}
}

func repairLock(opts *LockOptions) {
	if opts.isBlock && opts.blockWaitingSeconds <= 0 {
		opts.blockWaitingSeconds = DefaultBlockSeconds
	}
	if opts.expireSeconds <= 0 {
		opts.expireSeconds = DefaultLockExpireSeconds
	}
}
