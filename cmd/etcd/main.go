package main

import (
	"context"
	"fmt"
	"log"
	"time"

	//"go.etcd.io/etcd/clientv3"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	resp, err := cli.Grant(context.TODO(), 5)
	if err != nil {
		log.Fatal(err)
	}
	_, err = cli.Put(context.TODO(), "/services/book/127.0.0.1:8088", "bar", clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
	// the key 'foo' will be kept forever
	ch, err := cli.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		ka := <-ch
		fmt.Printf("ka ID:%d, ttl:%d\n", ka.ID, ka.TTL)
	}()
	time.Sleep(time.Second * 3)
	fmt.Printf("close lease\n")
	cli.Lease.Close()
	time.Sleep(time.Hour)
}
