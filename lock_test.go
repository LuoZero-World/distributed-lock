package lock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func Test_nonblockLock(t *testing.T) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	// 实际使用分布式锁中万万不可直接通过rdb删除 应该使用Unlock
	defer rdb.Del(ctx, "redis:lock:test1")

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

	wg.Wait() //协程1加上锁了 但不释放等待自然超时
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
	defer rdb.Del(ctx, "redis:lock:test1")

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

		lock2 := NewRedisLock("test1", rdb, WithExpireSeconds(2), WithBlock(), WithBlockWaitingSecond(5))
		if err := lock2.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()
}

func Test_lockWithWatchDog(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	defer rdb.Del(ctx, "redis:lock:test1")

	var wg sync.WaitGroup
	wg.Add(1)

	// 协程1加锁
	go func() {
		defer wg.Done()

		lock1 := NewRedisLock("test1", rdb, WithExpireSeconds(10), WithWatchDogMode())
		if err := lock1.Lock(ctx); err != nil {
			t.Error(err)
			return
		}

		// 模拟长时间的逻辑处理
		time.Sleep(15 * time.Second)

		if err := lock1.Unlock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	time.Sleep(time.Second * 2) //确保协程1加上了锁 但协程1、2要同时运行
	wg.Add(1)

	//协程2对锁操作
	go func() {
		defer wg.Done()

		lock2 := NewRedisLock("test1", rdb, WithBlock(), WithBlockWaitingSecond(30))
		if err := lock2.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()
}
