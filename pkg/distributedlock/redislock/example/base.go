package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/jursonmo/practise/pkg/distributedlock/redislock"
)

/*
redis-cli -h 192.168.64.5 -p 6379
192.168.64.5:6379> get dislock_key
"37ccbd6f-482b-4cba-86b0-b5b80a075e94"

192.168.64.5:6379> pttl dislock_key
(integer) 11084
*/
func main() {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "192.168.64.5:6379",
	})
	defer client.Close()

	dislock := redislock.NewDisLock(client, "dislock_key")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	ok, err := dislock.Lock(ctx, time.Second*8)
	if err != nil {
		log.Fatalln(err)
	}
	if !ok {
		log.Fatalln(err)
	}

	log.Println(dislock)

	exec_num := 0
	task := func(ctx context.Context) error {
		intv := time.Second * 5
		timer := time.NewTimer(intv)
		for {
			select {
			case <-ctx.Done():
				log.Println(ctx.Err())
				return ctx.Err()
			case <-timer.C:
				exec_num++
				if exec_num >= 3 {
					log.Printf("exec_num:%d, success and return\n", exec_num)
					return nil
				}
				log.Printf("task execute num:%d, It is not achievement \n", exec_num)
				return fmt.Errorf("exec_num:%d", exec_num)
			}
		}
	}

	err = dislock.Run(ctx, task)
	if err != nil {
		log.Println(err)
		return
	}
}
