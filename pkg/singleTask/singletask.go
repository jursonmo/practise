package singletask

import (
	"context"
	"sync"
	"time"
)

// singletask make sure there is only one task on working at once
// put new task will cancel last task and then run the new task
const (
	resultKey = "result"
)

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

//type TaskFunc func(context.Context) interface{}
//type TaskFunc func(ctx context.Context, args ...interface{}) error
type TaskFunc func(context.Context) error
type TaskResultHandler func(interface{})

func New(ctx context.Context) *SingleTask {
	ctx, cancel := context.WithCancel(ctx)
	return &SingleTask{ctx: ctx, cancel: cancel, resultCh: make(chan interface{}, 1)}
}

func (st *SingleTask) Close() {
	st.Lock()
	defer st.Unlock()
	if st.cancel != nil {
		st.cancel()
	}
}

//close singetask and wait task quit
func (st *SingleTask) CloseAndWait() {
	st.Lock()
	defer st.Unlock()
	if st.cancel != nil {
		st.cancel()
		st.CancelTask()
	}
}

func (st *SingleTask) CancelTask() {
	if st.taskCancel != nil {
		st.taskCancel()
		//wait to get cancel result
		result := getTaskResult(st.taskCtx)
		for _, resultHandler := range st.resultHandlers {
			resultHandler(result)
		}
		st.taskCancel = nil
		st.resultHandlers = nil
	}
}

// resultHandlers will be invoked each time when f return
func (st *SingleTask) PutTask(f TaskFunc, resultHandlers ...TaskResultHandler) error {
	return st.putTask(f, resultHandlers...)
}

//PutTaskPromise: if f return a non-nil err, means f fail, will retry
//intvl: call f interval time at least
//resultHandlers will be invoked each time when f return
func (st *SingleTask) PutTaskPromise(f TaskFunc, intvl time.Duration, resultHandlers ...TaskResultHandler) error {
	promiseFunc := func(ctx context.Context) error {
		if st.promise != nil {
			st.promise.Reset(ctx, NewRegularBackoff(intvl))
		} else {
			st.promise = NewPromise(ctx, NewRegularBackoff(intvl), ContextErrs())
		}
		return st.promise.Call(f, resultHandlers...).Error()
	}
	return st.putTask(promiseFunc)
}

func (st *SingleTask) putTask(f TaskFunc, resultHandlers ...TaskResultHandler) error {
	st.Lock()
	defer st.Unlock()

	//check if singTask is closed
	if err := st.ctx.Err(); err != nil {
		return err
	}

	//try to cancel the last task
	st.CancelTask()

	st.resultHandlers = resultHandlers
	st.taskCtx, st.taskCancel = witchCancelResult(st.ctx, st.resultCh)
	//用参数传入st.taskCtx, 确保goroutine func 运行时，f 用的是当前指定的st.taskCtx, 如果是闭包，有可能f 用的是后来新创建st.taskCtx
	go func(ctx context.Context) {
		result := f(ctx)
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
