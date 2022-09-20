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
var key = "distributed_lock_unit_test"

func TestBaseFunction(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redis_server,
	})
	defer client.Close()

	lock1 := NewDisLock(client, key)
	if lock1 == nil {
		t.Fatal("")
	}

	lock2 := NewDisLock(client, key)
	if lock2 == nil {
		t.Fatal("")
	}

	lock1TTL := time.Second * 2
	ok, err := lock1.Lock(context.Background(), lock1TTL)
	if err != nil || !ok {
		t.Fatal(err, ", lock1 get the distributed lock fail")
	}

	//lock2 can't aquire the distributed lock
	ok, err = lock2.Lock(context.Background(), lock1TTL/2)
	if err == nil || ok {
		t.Fatal(err, ", unexpected: lock2 aquire the distributed lock")
	}

	//lock1 release the distributed lock
	err = lock1.Unlock(context.Background())
	if err != nil {
		t.Fatal(err, ", lock1 unlock fail")
	}

	//now lock2 can aquire the distributed lock
	lock2TTL := time.Second
	ok, err = lock2.Lock(context.Background(), lock2TTL)
	if err != nil || !ok {
		t.Fatal(err, ", lock2: get the distributed lock fail")
	}

	//now lock1 can't aquire the distributed lock
	ok, err = lock1.Lock(context.Background(), lock2TTL/2)
	if err == nil || ok {
		t.Fatal(err, ", unexpected: lock1 aquire the distributed lock")
	}

	//lock2 release the distributed lock
	err = lock2.Unlock(context.Background())
	if err != nil {
		t.Fatal(err, ", lock2 unlock fail")
	}

	//test ttl after release the distributed lock, should return ttl 0
	ttl, err := lock2.TTL(context.Background())
	if ttl != 0 && err != nil {
		t.Fatal(err, ", lock2 unlock fail")
	}
}

func TestCompeteLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redis_server,
	})
	defer client.Close()

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
			timer := time.NewTimer(time.Millisecond * 500)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				t.Logf("task1")
				return errors.New("")
			}
			return errors.New("")
		}
		l1.Run(context.Background(), task1)
	}()

	//simulate another app to obtain distributed lock
	app2 := func() {
		defer wg.Done()
		l2 := NewDisLock(client, key)
		if l2 == nil {
			t.Fatal("")
		}

		ok, _ := l2.Lock(context.Background(), ttl-time.Millisecond*100)
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

func TestAutoRefresh(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redis_server,
	})
	defer client.Close()

	l1 := NewDisLock(client, key, WithBackoff(NonBackoff))
	if l1 == nil {
		t.Fatal("")
	}

	ttl := time.Second
	taskDeadline := time.Second * 3
	ctx, cancel := context.WithTimeout(context.Background(), taskDeadline)
	defer cancel()

	ok, err := l1.Lock(ctx, ttl)
	if err != nil || !ok {
		t.Fatal(err)
	}

	//simulate another app to obtain distributed lock
	app2 := func() {
		l2 := NewDisLock(client, key)
		if l2 == nil {
			t.Fatal("")
		}

		// lock2 aquire lock deadline is taskDeadline-time.Millisecond*10, bigger than ttl
		getLockDeadline := taskDeadline - time.Millisecond*10
		if getLockDeadline <= ttl {
			t.Fatal("getLockDeadline <= ttl")
		}
		//getLockDeadline is bigger than ttl , if lock2 can't aquire lock, it means auto refresh the distributed lock successfully
		ok, _ := l2.Lock(context.Background(), getLockDeadline)
		if ok {
			t.Fatal("unexpert, get lock")
		}

	}
	go app2()

	taskExecNum := 0
	taskIntv := time.Millisecond * 500
	//task never be executed successfully, so task will continue and lock will refresh until ctx deadline
	task := func(ctx context.Context) error {
		timer := time.NewTimer(taskIntv)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			t.Logf("task")
			taskExecNum++
			return errors.New("")
		}
		return errors.New("")
	}
	l1.Run(ctx, task)

	taskRunTime := taskIntv * time.Duration(taskExecNum)
	if taskRunTime < ttl {
		t.Fatalf(" taskRunTime(%v) < ttl(%v), auto refresh fail?", taskRunTime, ttl)
	}
}
