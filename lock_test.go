package lock

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
)

func Test_nonblockLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)

	// 协程1加锁
	go func() {
		defer wg.Done()

		lock1 := NewRedisLock("test1", rdb, WithExpireSeconds(2))
		if err := lock1.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()
	wg.Add(1)

	//协程2对锁操作
	go func() {
		defer wg.Done()

		lock2 := NewRedisLock("test1", rdb, WithExpireSeconds(2))
		if err := lock2.Unlock(ctx); err == nil || !errors.Is(err, ErrUnlockWithoutOwnership) {
			t.Errorf("got err: %v, expect: %v", err, ErrLockAcquiredByOthers)
			return
		}
	}()

	wg.Wait()
}

func Test_blockLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)

	// 协程1加锁
	go func() {
		defer wg.Done()

		lock1 := NewRedisLock("test1", rdb, WithExpireSeconds(2))
		if err := lock1.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()
	wg.Add(1)

	//协程2对锁操作
	go func() {
		defer wg.Done()

		lock2 := NewRedisLock("test1", rdb, WithBlock(), WithBlockWaitingSecond(5))
		if err := lock2.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()
}
