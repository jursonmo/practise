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
		return nil
	})
	taskMgr.Go("tasksleep2", func(ctx context.Context) error {
		time.Sleep(time.Second * 2)
		log.Println("tasksleep2 finished")
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
	log.Println(err)

	log.Println("finished tasks:", taskMgr.FinishedTasksName())

	log.Printf("unfinished tasksState:%+v", taskMgr.UnfinishedTasksState())
	log.Printf("finished tasksState:%+v", taskMgr.FinishedTasksState())
}
