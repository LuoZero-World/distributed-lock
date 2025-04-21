# distributed-lock
基于go-redis实现的简易分布式锁
- 支持阻塞轮询模式和非阻塞模式
- 支持看门狗模式对锁自动延期
- 实现了简易的红锁机制，然而RedLock在Redission中已经被标注为“过时”

## 快速开始

### 安装
```bash
go get github.com/LuoZero-World/distributed-lock
```

### 使用Demo
```go
package main

import (
	"context"
	"fmt"
	redislock "github.com/LuoZero-World/distributed-lock"
	"github.com/redis/go-redis/v9"
)

func main() {
    // 创建 Redis 客户端
    redisClient := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    // 创建一个上下文，用于取消锁操作
    ctx := context.Background()

    // 创建 RedisLock 对象，过期时间设置为2s
    lock := redislock.NewRedisLock("test1", redisClient, WithExpireSeconds(2))

    // 获取锁
    err := lock.Lock(ctx)
    if err != nil {
        fmt.Println("锁获取失败：", err)
        return
    }
    defer lock.Unlock(ctx) // 解锁

    // 在锁定期间执行任务
    // ...

    fmt.Println("任务执行完成")
}
```

## 其他用法

### 阻塞轮询
轮询是一种在锁可用之前反复获取锁的方式，可以使用 `WithBlock()` 选项来开启阻塞轮询，并通过`WithBlockWaitingSecond(5)`设置轮询的超时时间
```go
lock := redislock.NewRedisLock("test1", redisClient,
	WithBlock(), WithBlockWaitingSecond(5)
)
```

### 自动续期
可以通过使用 `WithWatchDogMode()` 选项在获取锁时启用看门狗功能：
```go
lock := redislock.NewRedisLock("test1", redisClient,
	WithWatchDogMode()
)
```
