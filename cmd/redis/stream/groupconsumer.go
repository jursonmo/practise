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

//xrevrange mystreamx + - count 1; xread block 0 streams mystreamx $
//xread: Any other options must come before the STREAMS option.
//xadd时，多个消费者xread可以同时读到最新消息。所以可以不需要组的概念。
// xack , xpending 都是基于组消费xreadgroup的概念,  xread 不需要xack.

//消费组存在在意义是什么： 主要是在把消息负载均衡到组内的多个消费者，提升消费能力，可以简单快速实现消费能力的横向扩展。
/*
https://redis.io/docs/data-types/streams/#consumer-groups

When the task at hand is to consume the same stream from different clients,
then XREAD already offers a way to fan-out to N clients, potentially also using replicas in order to provide more read scalability.
However in certain problems what we want to do is not to provide the same stream of messages to many clients,
 but to provide a different subset of messages from the same stream to many clients.
 An obvious case where this is useful is that of messages which are slow to process:
 the ability to have N different workers that will receive different parts of the stream allows us to scale message processing.
*/

// 如果组内消费者读到消息，没有ack,就会一直存在pending里，消费者下次消费时，可以指定从0开始，server 会把pending 的消息发给消费者
// 即消费者还是能接受到之前已经接受过的消息
// 如果消费者ack消息，就会把消息从pending里删除，即使消费者下次消费时，可以指定从0开始，也读不到此消息。
// ">",表示消费者从 server 最后发送给消费者的 group_last_delivered_id 消息开始接受消息。
// 也就是说，组消费者读取消息时，server 会把消息发给消费者，同时把消息放到消费者对应的pending里, 并且记录已经发给消费者的group_last_delivered_id
// 该组内的其他消费者只能从group_last_delivered_id开始消费，即一个消息不会发给组内的不同消费者。
// 消费组消费时，还有一个必须要考虑的问题，就是若某个消费者，消费了某条消息，但是并没有成功ack时（例如消费者进程宕机），
// 这条消息可能会丢失，因为组内其他消费者不能再次消费到该消息了。
// 消费者如果用 ">" 来拉取消息，就会发送比group_last_delivered_id还新的消息给消费者;
// 如果消费者指定 msgid 来消费，只能从pending里取出来发给消费者。比如指定0，那么就读取pending 的所有消息

/*
https://redis.io/docs/data-types/streams/#consumer-groups

If the ID is the special ID > then the command will return only new messages never delivered to other consumers so far,
 and as a side effect, will update the consumer group's last ID.
If the ID is any other valid numerical ID, then the command will let us access our history of pending messages.
 That is, the set of messages that were delivered to this specified consumer (identified by the provided name),
 and never acknowledged so far with XACK.
We can test this behavior immediately specifying an ID of 0, without any COUNT option, we'll just see the only pending message
()
However, if we acknowledge the message as processed, it will no longer be part of the pending messages history.
(如果ack 某个消息, 该消息就不会存在于pending 里，也就是从pending里删除)
*/
// xack , xpending 都是基于组消费xreadgroup的概念,  xread 不需要xack.

// ./groupconsumer -product=false -start="0" -ack=true , 每次从pending 的最小消息id开始读，并ack.  这样就可以删除pending 的所有消息。
// 命令行: XPENDING mystreamx ConsumerGroup1 - + 20 consumer1,  xpending 关联组内的消费者，即每个消费者的pending是不一样的
// 如果不指定消费者，那么就是显示组内所有消费者的pending 消息
// 127.0.0.1:6379> XPENDING mystreamx ConsumerGroup1 - + 20
// 1) 1) "1699416370752-0"
//    2) "consumer1"
//    3) (integer) 17796
//    4) (integer) 1
// 2) 1) "1699417020190-0"
//    2) "consumer2"
//    3) (integer) 17795
//    4) (integer) 1

func main() {
	flag.Parse()

	// Initialize the Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	{
		//test
		_, err := redisClient.XRevRangeN(ctx, "mystream1", "+", "-", 1).Result()
		// 没有stream消息, 不认为是错误, err 为 nil
		if err != nil {
			fmt.Printf("XRevRangeN err:%v", err)
		}
		return
	}

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
			Streams: []string{streamName, *start},        //如果*start是“0”，表示从pending 里读消息，读到的消息后必须ack,否则一直循环重复读pending的第一条消息
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
