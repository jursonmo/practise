package singletask

import (
	"context"
	"sync"
)

// singletask make sure there is only one task on working at once
// put new task will cancel last task and then run the new task
const (
	resultKey = "result"
)

type SingleTask struct {
	sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	taskCtx        context.Context
	taskCancel     context.CancelFunc
	resultHandlers []TaskResultHandler
}

//type TaskFunc func(context.Context) interface{}
type TaskFunc func(context.Context) error
type TaskResultHandler func(interface{})

func New(ctx context.Context) *SingleTask {
	ctx, cancel := context.WithCancel(ctx)
	return &SingleTask{ctx: ctx, cancel: cancel}
}

func (st *SingleTask) Close() {
	st.Lock()
	defer st.Unlock()
	if st.cancel != nil {
		st.cancel()
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

func (st *SingleTask) PutTask(f TaskFunc, resultHandlers ...TaskResultHandler) error {
	st.Lock()
	defer st.Unlock()

	//check if singTask is closed
	if err := st.ctx.Err(); err != nil {
		return err
	}

	//try to cancel the last task
	st.CancelTask()

	st.resultHandlers = resultHandlers
	st.taskCtx, st.taskCancel = witchCancelResult(st.ctx)
	go func() {
		result := f(st.taskCtx)
		//put result
		putTaskResult(st.taskCtx, result)
	}()

	return nil
}

func witchCancelResult(ctx context.Context) (context.Context, context.CancelFunc) {
	nctx, cancel := context.WithCancel(ctx)
	nctx = context.WithValue(nctx, resultKey, make(chan interface{}, 1))
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
