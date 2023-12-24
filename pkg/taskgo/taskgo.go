package taskgo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	TaskAdding = 1
	Stoped     = 2
)

// 一个任务也容易就起几个goroutine去完成, 但是这个stop 这个任务，需要知道哪些goroutine已经
// 处理完成，哪些没有处理完成，不然你可能就有goroutine泄露, 我们不能等待goroutine多到影响业务的
// 时候从用pprof去查看，那时太晚了，而且不容易快速查出问题，
// 应该是再结束一个任务时，就要保证其下的goroutine的能正常地在规定的时间
// 退出,否则就打印error, 开发人员提前去查问题。
type TaskGo struct {
	ctx       context.Context
	cancel    context.CancelFunc //取消任务时默认都调用context.CancelFunc
	canceFunc func()             //用户自定义自己取消任务的handler,
	//wg       sync.WaitGroup
	//tasks    sync.Map
	tasks map[string]*Result
	sync.Mutex
	status   int32
	tasksNum int32
	doneCh   chan struct{}
}
type Result struct {
	TaskName string
	DoneAt   time.Time
	Err      error
}

func NewTaskGo(ctx context.Context) *TaskGo {
	tg := &TaskGo{doneCh: make(chan struct{}, 1)}
	tg.ctx, tg.cancel = context.WithCancel(ctx)
	return tg
}

func (tg *TaskGo) SetCancelFunc(f func()) {
	tg.canceFunc = f
}

func (tg *TaskGo) Stop() {
	tg.Lock()
	defer tg.Unlock()
	tg.status = Stoped
}

func (tg *TaskGo) IsStoped() bool {
	tg.Lock()
	defer tg.Unlock()
	return tg.isStoped()
}

func (tg *TaskGo) isStoped() bool {
	return tg.status == Stoped
}

// func (tg *TaskGo) incr() {
// 	atomic.AddInt32(&tg.tasksNum, 1)
// }

// func (tg *TaskGo) decr() {
// 	atomic.AddInt32(&tg.tasksNum, -1)
// }

func (tg *TaskGo) Go(name string, f func(ctx context.Context) error) error {
	tg.Lock()
	defer tg.Unlock()

	if tg.isStoped() {
		return errors.New("stoped")
	}

	_, b := tg.tasks[name]
	if b {
		return fmt.Errorf("task:%s already running", name)
	}
	r := &Result{TaskName: name}
	tg.tasks[name] = r
	tg.tasksNum += 1

	go func() {
		err := f(tg.ctx)
		tg.done(r, err)
	}()
	return nil
}

func (tg *TaskGo) done(r *Result, err error) {
	tg.Lock()
	defer tg.Unlock()
	//log.Printf("goroutine:%v finish\n", r.TaskName)
	r.Err = err
	r.DoneAt = time.Now()

	tg.tasksNum -= 1
	if tg.tasksNum < 0 {
		panic("tasksNum < 0, never happend")
	}
	//由于每次拉起goroutine之前都会检查是否stop
	//如果关掉，并且目前tasksNum为0，说明不可能有goroutine再运行了
	if tg.isStoped() && tg.tasksNum == 0 {
		tg.doneCh <- struct{}{}
	}
}

func (tg *TaskGo) unfinishTasks() []string {
	tg.Lock()
	defer tg.Unlock()
	tasks := make([]string, 0, len(tg.tasks))
	for name, r := range tg.tasks {
		if r.DoneAt.IsZero() {
			tasks = append(tasks, name)
		}
	}
	return tasks
}

func (tg *TaskGo) StopAndWait(d time.Duration) error {
	if tg.IsStoped() {
		return errors.New("stoped")
	}
	tg.Stop()
	tg.cancel()
	if tg.canceFunc != nil {
		tg.canceFunc()
	}
	select {
	case <-time.After(d):
		//stop的期限到了，goroutine没有全部退出，把没有退出的goroutine 输出
		tasks := tg.unfinishTasks()
		return fmt.Errorf("unfinish tasks:%v", tasks)
	case <-tg.doneCh:
		//task下的所有goroutine都已经退出了
		return nil
	}
	return nil
}
