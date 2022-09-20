package main

import (
	"context"
	"errors"
	"log"
	"sync"
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

	l1 := redislock.NewDisLock(client, key)
	if l1 == nil {
		log.Fatal("")
	}

	ttl := time.Second * 2

	ok, err := l1.Lock(context.Background(), ttl)
	if err != nil || !ok {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		task1 := func(ctx context.Context) error {
			timer := time.NewTimer(time.Millisecond * 500)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				log.Println("task1")
				return errors.New("")
			}
			return errors.New("")
		}
		err = l1.Run(context.Background(), task1)
		if err != nil {
			log.Fatal(err)
		}
	}()

	//simulate another app to obtain distributed lock
	app2 := func() {
		defer wg.Done()
		l2 := redislock.NewDisLock(client, key)
		if l2 == nil {
			log.Fatal("")
		}

		ok, _ := l2.Lock(context.Background(), ttl-time.Millisecond*100)
		if ok {
			log.Fatal("unexpert, get lock")
		}

	}
	go app2()

	wg.Wait()

	//now we should get lock success

	l3 := redislock.NewDisLock(client, key)
	if l3 == nil {
		log.Fatal("")
	}

	ok, err = l3.Lock(context.Background(), ttl-time.Millisecond*10)
	if err != nil || !ok {
		log.Fatal(err, "we should get lock success")
	}

	err = l3.Release(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
