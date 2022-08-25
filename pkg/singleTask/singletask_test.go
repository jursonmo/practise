package singletask

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var taskCounter int32

func TestSingleTask(t *testing.T) {
	myTask := func(ctx context.Context) error {
		new := atomic.AddInt32(&taskCounter, 1)
		defer atomic.AddInt32(&taskCounter, -1)
		if new > 1 {
			err := fmt.Errorf("there is too many task on working, expect only on task")
			t.Fatal(err)
		}
		timer := time.NewTimer(time.Second * 1)
		defer timer.Stop()
		select {
		case <-timer.C:
			t.Log("work time over")
			return nil
		case <-ctx.Done():
			t.Logf("task cancel? err:%v\n", ctx.Err())
			return nil
		}
		return nil
	}
	myTaskResultHandler := func(result interface{}) {
		if err, ok := result.(error); ok {
			t.Fatal(err)
		}
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	st := New(ctx)
	for i := 0; i < 5; i++ {
		err = st.PutTask(myTask, myTaskResultHandler)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * 2)
		err = st.PutTask(myTask, myTaskResultHandler)
		if err != nil {
			t.Fatal(err)
		}
	}

	cancel() //st.Close(); cancel the singletask, make the singletask been closed
	// _ = cancel
	// st.Close()
	err = st.PutTask(myTask, myTaskResultHandler)
	if err == nil {
		t.Fatal("expect err when putting  new Task to the closed singletask ")
	}
}
func TestCloseAndWait(t *testing.T) {
	var err error
	ctx, _ := context.WithCancel(context.Background())
	st := New(ctx)

	taskOver := false
	myTask := func(ctx context.Context) error {
		defer func() { taskOver = true }()
		time.Sleep(time.Second * 2)
		return nil
	}
	err = st.PutTask(myTask)
	if err != nil {
		t.Fatalf("put task, unexpect")
	}
	st.CloseAndWait()
	if !taskOver {
		t.Fatalf("CloseAndWait don't wait mytask finish")
	}
}

/*
func myTask(ctx context.Context) error {
	defer atomic.AddInt32(&taskCounter, -1)
	new := atomic.AddInt32(&taskCounter, 1)
	if new > 1 {
		err := fmt.Errorf("there is too many task on working, expect only on task")
		panic(err)
		return err
	}
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()
	select {
	case <-timer.C:
		fmt.Println("work time over")
		return nil
	case <-ctx.Done():
		fmt.Printf("cancel?:%v\n", ctx.Err())
		return nil
	}
	return nil
}

func myTaskFail(result interface{}) {
	if err, ok := result.(error); ok {
		t.Fatal(err)
	}
}
*/
