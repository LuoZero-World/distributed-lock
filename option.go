package lock

const (
	// 默认的阻塞等待时间 s
	DefaultBlockSeconds = 5
	// 默认的分布式锁过期时间 s
	DefaultLockExpireSeconds = 30

	//看门狗工作时间步 s
	DefaultWatchDogWorkStepSeconds = 5
)

type LockOption func(*LockOptions)

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
