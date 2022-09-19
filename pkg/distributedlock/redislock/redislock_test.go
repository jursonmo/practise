package redislock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v9"
)

var redis_server = "192.168.64.5:6379"

func TestAbtainLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redis_server,
	})
	defer client.Close()

	key := "distributed_key"

	l1 := NewDisLock(client, key)
	if l1 == nil {
		t.Fatal("")
	}

	ttl := time.Second * 2

	ok, err := l1.Lock(context.Background(), ttl)
	if err != nil || !ok {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		task1 := func(ctx context.Context) error {
			timer := time.NewTimer(ttl - time.Millisecond*20)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				t.Logf("task1")
				return nil
			}
			return errors.New("")
		}
		err = l1.Run(context.Background(), task1)
		if err != nil {
			t.Fatal(err)
		}
	}()

	//simulate another app to obtain distributed lock
	app2 := func() {
		defer wg.Done()
		l2 := NewDisLock(client, key)
		if l2 == nil {
			t.Fatal("")
		}

		ok, _ := l2.Lock(context.Background(), ttl-time.Millisecond*10)
		if ok {
			t.Fatal("unexpert, get lock")
		}

	}
	go app2()

	wg.Wait()

	//now we should get lock success

	l3 := NewDisLock(client, key)
	if l3 == nil {
		t.Fatal("")
	}

	ok, err = l3.Lock(context.Background(), ttl-time.Millisecond*10)
	if err != nil || !ok {
		t.Fatal(err, "we should get lock success")
	}

	err = l3.Release(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}
