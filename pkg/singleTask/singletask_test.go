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
		t.Fatalf("put task err, unexpect")
	}
	st.CloseAndWait()
	if !taskOver {
		t.Fatalf("unexpect: CloseAndWait api don't wait task finish")
	}
}

func TestCancelTask(t *testing.T) {
	var err error
	ctx, _ := context.WithCancel(context.Background())
	st := New(ctx)

	taskCancelTimes := 0
	myTask := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			//cancel task, count times
			taskCancelTimes++
			return ctx.Err()
		case <-time.After(time.Second):
			//simulate work spend one Second
			return nil //return nil means task finish ok
		}
	}
	err = st.PutTask(myTask)
	if err != nil {
		t.Fatalf("put task err, unexpect")
	}
	st.CancelTask()
	if taskCancelTimes != 1 {
		t.Fatalf("unexpect: CancelTask api don't wait task finish")
	}

	//singleTask can be put a other new task
	err = st.PutTask(myTask)
	if err != nil {
		t.Fatalf("put task err, unexpect")
	}
	st.CancelTask() //cancel task again
	if taskCancelTimes != 2 {
		t.Fatalf("unexpect: CancelTask() api can't cancel task again")
	}
}

func TestPutTaskPromise(t *testing.T) {
	var err error
	ctx, _ := context.WithCancel(context.Background())
	st := New(ctx)

	myTaskCall := 0
	needCall := 3
	myTask := func(ctx context.Context) error {
		myTaskCall++
		//simulate do some work
		time.Sleep(time.Millisecond * 100)
		t.Logf("myTask call:%d", myTaskCall)
		if myTaskCall < needCall {
			//return err, means myTask executed fail, will be try again
			return fmt.Errorf("fail,try again")
		}

		//if myTaskCall == needCall, return nil means myTask executed successfully, don't try again
		return nil
	}
	myTaskResultHandler := func(result interface{}) {
		if err, ok := result.(error); ok {
			t.Log(err)
		}
	}

	err = st.PutTaskPromise(myTask, time.Millisecond*100, myTaskResultHandler)
	if err != nil {
		t.Fatalf("put task err, unexpect")
	}

	time.Sleep(time.Second)
	if myTaskCall != needCall {
		t.Fatalf("myTaskCall != needCall")
	}
}
