package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client
var ctx = context.Background()

var (
	productEnable = flag.Bool("product", true, "product new message")
	start         = flag.String("start", ">", "consumer start from ?")
	ackEnable     = flag.Bool("ack", false, "ack message")
	count         = flag.Int("count", 1, "max messages on one read call")
	block         = flag.Int("block", 0, "seconds timeout for one read call")
)

// 如果消费者读到消息，没有ack,就会一直存在pending里，消费者下次消费时，可以指定从0开始，server 会把pending 的消息发给消费者
// 即消费者还是能接受到之前已经接受过的消息
// 如果消费者ack消息，就会把消息从pending里删除，即使消费者下次消费时，可以指定从0开始，也读不到此消息。
// ">",表示消费者从 server 最后发送给消费者的 group_last_delivered_id 消息开始接受消息。
// 也就是说，组消费者读取消息时，server 会把消息发给消费者，同时把消息放到消费者对应的pending里, 并且记录已经发给消费者的group_last_delivered_id
// 该组内的其他消费者只能从group_last_delivered_id开始消费，即一个消息不会发给组内的不同消费者。
// 消费组消费时，还有一个必须要考虑的问题，就是若某个消费者，消费了某条消息，但是并没有成功ack时（例如消费者进程宕机），
// 这条消息可能会丢失，因为组内其他消费者不能再次消费到该消息了。
// 消费者如果用 ">" 来拉取消息，就会发送比group_last_delivered_id还新的消息给消费者; 如果消费者指定 msgid 来消费，只能从pending里取出来发给消费者。
// ./groupconsumer -product=false -start="0" -ack=true , 每次从pending 的最小消息id开始读，并ack.  这样就可以删除pending 的所有消息。
// 命令行: XPENDING mystreamx ConsumerGroup1 - + 20 consumer1,  xpending 关联组内的消费者，即每个消费者的pending是不一样的
func main() {
	flag.Parse()

	// Initialize the Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create a stream with a few sample messages
	streamName := "mystreamx"

	// Consumer 1 in Group 1
	group1 := "ConsumerGroup1"
	_, err := redisClient.XGroupCreateMkStream(ctx, streamName, group1, "0").Result()
	if err != nil {
		//如果组已经存在，打印Failed to create Consumer Group 1: BUSYGROUP Consumer Group name already exists
		log.Printf("Failed to create Consumer Group 1: %v", err)
	}

	// Consumer 2 in Group 2
	group2 := "ConsumerGroup2"
	_, err = redisClient.XGroupCreateMkStream(ctx, streamName, group2, "0").Result()
	if err != nil {
		log.Printf("Failed to create Consumer Group 2: %v", err)
	}

	//是否发送
	if *productEnable {
		messages := []string{"Message 1", "Message 2", "Message 3"}
		for i, msg := range messages {
			_, err := redisClient.XAdd(ctx, &redis.XAddArgs{
				Stream: streamName,
				Values: map[string]interface{}{"message": msg},
			}).Result()
			if err != nil {
				log.Fatalf("Failed to add message %d to stream: %v", i, err)
			}

		}
	}

	// Start the consumers
	go consumeMessages(streamName, group1, "consumer1")
	go consumeMessages(streamName, group2, "consumer2")

	// Keep the program running
	select {}
}

func consumeMessages(streamName, groupName, consumerID string) {
	readcall := 0
	for {
		messages, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerID,
			//Streams:  []string{streamName, ">"},
			Streams: []string{streamName, *start},
			Block:   time.Duration(*block) * time.Second, // 0:Wait indefinitely for new messages
			Count:   int64(*count),                       // Process message at a time
		}).Result()
		if err != nil {
			//超时打印: Consumer consumer1: Error reading from stream: redis: nil
			log.Printf("Consumer %s: Error reading from stream: %v", consumerID, err)
			continue
		}

		readcall++
		for _, message := range messages {
			for _, xMessage := range message.Messages {
				fmt.Printf("Consumer %s: read call:%d, Received message %s: %v\n", consumerID, readcall, xMessage.ID, xMessage.Values)
				if *ackEnable {
					redisClient.XAck(ctx, streamName, groupName, xMessage.ID)
				}
			}
		}
	}
}
