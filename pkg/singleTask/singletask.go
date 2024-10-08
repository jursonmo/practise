package singletask

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

// singletask make sure there is only one task on working at once
// put new task will cancel last task and then run the new task
const (
	resultKey = "result"
)

var TaskPanicError = errors.New("taskPanic")

type SingleTask struct {
	sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	resultCh chan interface{}

	taskCtx        context.Context
	taskCancel     context.CancelFunc
	resultHandlers []TaskResultHandler

	promise *Promise
}

// type TaskFunc func(context.Context) interface{}
// type TaskFunc func(ctx context.Context, args ...interface{}) error
type TaskFunc func(context.Context) error
type TaskResultHandler func(interface{}) //task result handler, should be unblock

func New(ctx context.Context) *SingleTask {
	ctx, cancel := context.WithCancel(ctx)
	return &SingleTask{ctx: ctx, cancel: cancel, resultCh: make(chan interface{}, 1)}
}

// close singetask, and the singleTask can't be put new task
func (st *SingleTask) Close() {
	st.Lock()
	defer st.Unlock()
	if st.cancel != nil {
		st.cancel()
	}
}

// close singetask and wait task quit, and the singleTask can't be put new task
func (st *SingleTask) CloseAndWait() {
	st.Lock()
	defer st.Unlock()
	if st.cancel != nil {
		st.cancel()
		st.cancelTask()
	}
}

// just cancel current running task and wait task end, the singelTask can still put a new task
func (st *SingleTask) CancelTask() {
	st.Lock()
	defer st.Unlock()
	st.cancelTask()
}

func (st *SingleTask) cancelTask() {
	if st.taskCancel != nil {
		st.taskCancel()
		//wait to task end and get the canceled task result
		getTaskResult(st.taskCtx)
		// result := getTaskResult(st.taskCtx)
		// for _, resultHandler := range st.resultHandlers {
		// 	resultHandler(result)
		// }
		st.taskCtx = nil
		st.taskCancel = nil
		st.resultHandlers = nil
	}
}

func (st *SingleTask) IsTaskRunning() bool {
	st.Lock()
	defer st.Unlock()

	//haven't started or have been canceled?
	if st.taskCtx == nil {
		return false
	}
	return !hasResult(st.taskCtx)
}

// resultHandlers will be invoked each time when f return
func (st *SingleTask) PutTask(f TaskFunc, resultHandlers ...TaskResultHandler) error {
	return st.putTask(f, false, resultHandlers...)
}

// resultHandlers will be invoked each time when f return; it will recover when f() panic
func (st *SingleTask) PutTaskSafe(f TaskFunc, resultHandlers ...TaskResultHandler) error {
	return st.putTask(f, true, resultHandlers...)
}

// PutTaskPromise: if f return a non-nil err, means f fail, will retry
// intvl: call f interval time at least
// resultHandlers will be invoked each time when f return
func (st *SingleTask) PutTaskPromise(f TaskFunc, intvl time.Duration, resultHandlers ...TaskResultHandler) error {
	promiseWrapFunc := func(ctx context.Context) error {
		if st.promise != nil {
			st.promise.Reset(ctx, nil)
		} else {
			st.promise = NewPromise(ctx, backoffx.NewLinearBackoff(intvl), ContextErrs())
		}
		return st.promise.Call(f, resultHandlers...).Error()
	}
	return st.putTask(promiseWrapFunc, false)
}

// runSafe means it will recover when f called panic
func (st *SingleTask) putTask(f TaskFunc, runSafe bool, resultHandlers ...TaskResultHandler) error {
	st.Lock()
	defer st.Unlock()

	//check if singTask is closed
	if err := st.ctx.Err(); err != nil {
		return err
	}

	//try to cancel the last task
	st.cancelTask()

	st.resultHandlers = resultHandlers
	st.taskCtx, st.taskCancel = witchCancelResult(st.ctx, st.resultCh)
	//用参数传入st.taskCtx, 确保goroutine func 运行时，f 用的是当前指定的st.taskCtx, 如果是闭包，有可能f 用的是后来新创建st.taskCtx
	go func(ctx context.Context) {
		if runSafe {
			var err error
			if r := recover(); r != nil {
				if v, ok := r.(error); ok {
					err = v
				} else {
					err = fmt.Errorf("recover:%v, err:%w", r, TaskPanicError)
				}
				for _, resultHandler := range st.resultHandlers {
					resultHandler(err)
				}
			}
		}
		result := f(ctx)
		for _, resultHandler := range st.resultHandlers {
			resultHandler(result)
		}
		//put result
		putTaskResult(ctx, result)
	}(st.taskCtx)

	return nil
}

func witchCancelResult(ctx context.Context, resultCh chan interface{}) (context.Context, context.CancelFunc) {
	nctx, cancel := context.WithCancel(ctx)
	//nctx = context.WithValue(nctx, resultKey, make(chan interface{}, 1))
	nctx = context.WithValue(nctx, resultKey, resultCh)
	return nctx, cancel
}

func getTaskResult(ctx context.Context) interface{} {
	ch := ctx.Value(resultKey).(chan interface{})
	return <-ch
}

func putTaskResult(ctx context.Context, result interface{}) {
	ch := ctx.Value(resultKey).(chan interface{})
	ch <- result
}

func hasResult(ctx context.Context) bool {
	ch := ctx.Value(resultKey).(chan interface{})
	return len(ch) > 0
}
