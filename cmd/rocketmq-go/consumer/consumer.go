package main

import (
	"context"
	"fmt"
	"os"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

//https://github.com/apache/rocketmq-client-go/blob/master/examples/consumer/simple/main.go
func main() {
	groupName := "route-group"
	topic := "node-router-topic"
	brokerAddr := "172.18.1.163:9876"
	sig := make(chan os.Signal)
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName(groupName),
		consumer.WithConsumerModel(consumer.Clustering), //=========BroadCasting
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{brokerAddr})),
	)
	err := c.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for i := range msgs {
			fmt.Printf("subscribe callback: %v \n", msgs[i])
		}

		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	// Note: start after subscribe
	err = c.Start()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	<-sig
	err = c.Shutdown()
	if err != nil {
		fmt.Printf("shutdown Consumer error: %s", err.Error())
	}
}
