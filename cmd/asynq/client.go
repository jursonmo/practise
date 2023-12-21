package main

import (
	"log"
	"time"

	"github.com/hibiken/asynq"
)

const redisAddr = "192.168.10.200:6379" //"192.168.64.5:6379"

func main() {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer client.Close()

	// ------------------------------------------------------
	// Example 1: Enqueue task to be processed immediately.
	//            Use (*Client).Enqueue method.
	// ------------------------------------------------------

	task, err := NewEmailDeliveryTask(43, "some:template:id")
	if err != nil {
		log.Fatalf("could not create task: %v", err)
	}
	info, err := client.Enqueue(task, asynq.Queue("low"), asynq.Timeout(10*time.Minute)) //这里的timeout是告诉处理者处理这个消息时ctx 的超时时间
	//info, err := client.Enqueue(task)
	if err != nil {
		log.Fatalf("could not enqueue task: %v", err)
	}
	log.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)

	// ------------------------------------------------------------
	// Example 2: Schedule task to be processed in the future.
	//            Use ProcessIn or ProcessAt option.
	// ------------------------------------------------------------
	task1, err := NewEmailDeliveryTask(44, "should handler in one minute")
	info, err = client.Enqueue(task1, asynq.ProcessIn(time.Minute)) //timeout 默认是30分钟吗1800秒
	if err != nil {
		log.Fatalf("could not schedule task: %v", err)
	}
	log.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)

	// ----------------------------------------------------------------------------
	// Example 3: Set other options to tune task processing behavior.
	//            Options include MaxRetry, Queue, Timeout, Deadline, Unique etc.
	// ----------------------------------------------------------------------------

	task, err = NewImageResizeTask("https://example.com/myassets/image.jpg")
	if err != nil {
		log.Fatalf("could not create task: %v", err)
	}
	info, err = client.Enqueue(task, asynq.MaxRetry(1), asynq.Timeout(2*time.Minute))
	if err != nil {
		log.Fatalf("could not enqueue task: %v", err)
	}
	log.Printf("enqueued task: id=%s queue=%s", info.ID, info.Queue)
}
