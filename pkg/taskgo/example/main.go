package main

import (
	"context"
	"log"
	"time"

	"github.com/jursonmo/practise/pkg/taskgo"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel

	taskMgr := taskgo.NewTaskGo(ctx)

	taskMgr.Go("tasksleep1", func(ctx context.Context) error {
		time.Sleep(time.Second)
		log.Println("tasksleep1 finished")
		return nil //这样在后面FinishedTasksState()打印那里，tasksleep1的Err为nil
	})
	taskMgr.GoSafe("tasksleep2", func(ctx context.Context) error {
		time.Sleep(time.Second * 2)
		log.Println("tasksleep2 finished")
		if true {
			//panic 返回一个字符串，这个panic 会被recover()捕获并封装成taskgo.PanicError
			log.Panic("panic_in_tasksleep2") //任务返回错误，这样在后面FinishedTasksState()打印那里，tasksleep2的Err不为空了
		} else {
			//主动制造一个除0的错误
			a := 0
			_ = 1 / a //如果执行这recover捕获到err,是 runtime error: integer divide by zero
		}
		return nil
	})

	taskMgr.Go("tasksleep4", func(ctx context.Context) error {
		time.Sleep(time.Second * 4)
		log.Println("tasksleep4 finished")
		return nil
	})
	taskMgr.Go("tasksleep5", func(ctx context.Context) error {
		time.Sleep(time.Second * 5)
		log.Println("tasksleep5 finished")
		return nil
	})

	//wait for tasksleep1 and tasksleep2 done
	time.Sleep(time.Second * 3)
	log.Println("stopping taskgo...")
	err := taskMgr.StopAndWait(time.Millisecond * 100)
	log.Println(err) //这里会打印没有结束的task name

	log.Println("finished tasks:", taskMgr.FinishedTasksName())
	log.Printf("finished tasksState:%+v\n", taskMgr.FinishedTasksState())

	//我们一般比较关注没有正常结束的任务，以及结束但是有错误的任务
	log.Printf("unfinished tasksState:%+v\n", taskMgr.UnfinishedTasksState())
	log.Printf("finished but has error, tasksState:%+v\n", taskMgr.ErrorTasksState()) //已经结束但是有错误的任务

}
