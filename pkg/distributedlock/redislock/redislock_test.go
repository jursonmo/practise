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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ok, err := l1.Lock(ctx, time.Second*2)
	if err != nil || !ok {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		task1 := func(ctx context.Context) error {
			timer := time.NewTimer(time.Millisecond * 1900)
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
		err = l1.Run(ctx, task1)
		if err != nil {
			t.Fatal(err)
		}
	}()

	//simulate another task to obtain distributed lock
	go func() {
		defer wg.Done()
		l2 := NewDisLock(client, key)
		if l2 == nil {
			t.Fatal("")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1900)
		defer cancel()

		ok, _ := l2.Lock(ctx, time.Millisecond*1900)
		if ok {
			t.Fatal("unexpert, get lock")
		}

	}()

	wg.Wait()
}
