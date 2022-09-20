package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/jursonmo/practise/pkg/distributedlock/redislock"
)

var redis_server = "192.168.64.5:6379"

func main() {

	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redis_server,
	})
	defer client.Close()

	key := "distributed_key"

	l1 := redislock.NewDisLock(client, key, redislock.WithBackoff(redislock.NonBackoff))
	if l1 == nil {
		log.Fatal("")
	}

	ttl := time.Second
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	ok, err := l1.Lock(ctx, ttl)
	if err != nil || !ok {
		log.Fatal(err)
	}

	taskExecNum := 0
	taskIntv := time.Millisecond * 500
	task := func(ctx context.Context) error {
		timer := time.NewTimer(taskIntv)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			log.Println("task")
			taskExecNum++
			return errors.New("")
		}
		return errors.New("")
	}
	l1.Run(ctx, task)

	taskRunTime := taskIntv * time.Duration(taskExecNum)
	if taskRunTime < ttl {
		log.Fatalf(" taskRunTime(%v) < ttl(%v), auto refresh fail?", taskRunTime, ttl)
	}
}
