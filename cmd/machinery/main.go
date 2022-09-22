package main

import (
	"log"

	machinery "github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
)

// SumInts ...
func SumInts(numbers []int64) (int64, error) {
	var sum int64
	for _, num := range numbers {
		sum += num
	}
	return sum, nil
}

func main() {

	cnf, err := config.NewFromYaml("./config.yml", false)
	if err != nil {
		log.Println("config failed", err)
		return
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		log.Println("start server failed", err)
		return
	}

	// 注册任务
	err = server.RegisterTask("sum", SumInts)
	if err != nil {
		log.Println("reg task failed", err)
		return
	}

	worker := server.NewWorker("asong", 1)
	go func() {
		// broker recevice msg, and woker is TaskProcessor that do with the msg
		//work 根据收到的任务信息，查看server是否注册对应的处理handler, 如果有则调用handler
		//并且把处理结果发送到result_backend,
		err = worker.Launch()
		if err != nil {
			log.Println("start worker error", err)
			return
		}
	}()

	//task signature
	signature := &tasks.Signature{
		Name: "sum",
		Args: []tasks.Arg{
			{
				Type:  "[]int64",
				Value: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
		},
	}

	asyncResult, err := server.SendTask(signature) //--> server.broker.publish msg to redis_broker, signature.UUID auto generate
	if err != nil {
		log.Fatal(err)
	}
	res, err := asyncResult.Get(1) // 从backend 读 结果，根据asyncResult.Signature.UUID 作为Key 去读取结果。
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("get res is %v\n", tasks.HumanReadableResults(res))

}
