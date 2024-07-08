package taskgo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	Stoped = 2
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
	tasks     map[string]*TaskState
	sync.Mutex
	status   int32
	tasksNum int32
	doneCh   chan struct{} // notfiy when all tasks are done
}

type TaskState struct {
	TaskName string
	StartAt  time.Time
	DoneAt   time.Time
	Err      error
}

func NewTaskGo(ctx context.Context) *TaskGo {
	tg := &TaskGo{tasks: make(map[string]*TaskState), doneCh: make(chan struct{}, 1)}
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

var PanicError = errors.New("TaskGoPanicError")

func (tg *TaskGo) GoSafe(taskName string, f func(ctx context.Context) error) error {
	//为了安全的运行f, panic 也不会导致整个进程挂掉, 包装下f, 加入recover, 当发送panic, 就返回一个err,底层是
	safeF := func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				//panic("mypanicTest"), recover()捕捉到是一个字符串;panic(errors.New("mypanicTest")),recover()捕捉到是一个error
				//如果recover 捕捉到的是一个error,那么就用这个error来返回, 否则用底层是PanicError 的error来代替
				if v, ok := r.(error); ok {
					err = v
					return
				}
				err = fmt.Errorf("recover:%v, err:%w", r, PanicError) //上层业务可以用errors.Is(err, PanicError)来判断
				return
			}
		}()
		return f(ctx)
	}
	return tg.Go(taskName, safeF)
}

func (tg *TaskGo) Go(taskName string, f func(ctx context.Context) error) error {
	tg.Lock()
	defer tg.Unlock()

	if tg.isStoped() {
		return errors.New("taskgo is stoped")
	}

	_, b := tg.tasks[taskName]
	if b {
		return fmt.Errorf("task:%s already running", taskName)
	}
	ts := &TaskState{TaskName: taskName, StartAt: time.Now()}
	tg.tasks[taskName] = ts
	tg.tasksNum += 1

	go func() {
		err := f(tg.ctx)
		tg.done(ts, err)
	}()
	return nil
}

func (tg *TaskGo) done(ts *TaskState, err error) {
	tg.Lock()
	defer tg.Unlock()
	//log.Printf("goroutine:%v finish\n", r.TaskName)
	ts.Err = err
	ts.DoneAt = time.Now()

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

func (ts *TaskState) unCompleted() bool {
	return ts.DoneAt.IsZero() //没有结束，DoneAt 为“0”
}

func (tg *TaskGo) UnfinishedTasksName() []string {
	return tg.iterTasks(func(ts *TaskState) bool {
		return ts.unCompleted()
	})
}

func (tg *TaskGo) FinishedTasksName() []string {
	return tg.iterTasks(func(ts *TaskState) bool {
		return !ts.unCompleted()
	})
}

func (tg *TaskGo) AllTasksName() []string {
	return tg.iterTasks(func(ts *TaskState) bool {
		return ts != nil
	})
}

func (tg *TaskGo) iterTasks(condition func(ts *TaskState) bool) []string {
	tg.Lock()
	defer tg.Unlock()
	tasks := make([]string, 0, len(tg.tasks))
	for name, ts := range tg.tasks {
		if condition(ts) {
			tasks = append(tasks, name)
		}
	}
	return tasks
}

func (tg *TaskGo) UnfinishedTasksState() []TaskState {
	return tg.iterTasksState(func(ts *TaskState) bool {
		return ts.unCompleted()
	})
}

func (tg *TaskGo) FinishedTasksState() []TaskState {
	return tg.iterTasksState(func(ts *TaskState) bool {
		return !ts.unCompleted()
	})
}

func (tg *TaskGo) ErrorTasksState() []TaskState {
	return tg.iterTasksState(func(ts *TaskState) bool {
		return !ts.unCompleted() && ts.Err != nil //结束了，但是有错误的任务
	})
}

func (tg *TaskGo) iterTasksState(condition func(ts *TaskState) bool) []TaskState {
	tg.Lock()
	defer tg.Unlock()
	tasks := make([]TaskState, 0, len(tg.tasks))
	for _, ts := range tg.tasks {
		if condition(ts) {
			tasks = append(tasks, *ts)
		}
	}
	return tasks
}

func (tg *TaskGo) StopAndWait(d time.Duration) error {
	if tg.IsStoped() {
		return errors.New("already stoped")
	}
	tg.Stop()
	tg.cancel()
	if tg.canceFunc != nil {
		tg.canceFunc()
	}
	select {
	case <-time.After(d):
		//stop的期限到了，goroutine没有全部退出，把没有退出的goroutine 输出
		tasks := tg.UnfinishedTasksName()
		return fmt.Errorf("stop all task err,unfinish tasks:%v", tasks)
	case <-tg.doneCh:
		//task下的所有goroutine都已经退出了
		return nil
	}
}
