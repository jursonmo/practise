package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func main() {
	//创建v3的client端，我们和etcd的所有接口都是通过该client实现
	ctx, _ := context.WithTimeout(context.TODO(), 10*time.Second)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:         []string{"127.0.0.1:2379", "192.168.64.5:2379"},
		DialTimeout:       5 * time.Second,
		DialKeepAliveTime: 5 * time.Second,
		Context:           ctx,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	fmt.Println("new cli ok")
	//创建两个竞争的Session，这里Session有效期使用默认值60s
	s1, err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s1.Close()
	fmt.Println("NewSession s1 ok")
	//pfx 是"/my-election/"
	e1 := concurrency.NewElection(s1, "/my-election/")
	fmt.Println("NewElection e1 ok")

	s2, err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s2.Close()
	//pfx 是"/my-election/"
	e2 := concurrency.NewElection(s2, "/my-election/")
	fmt.Println("NewElection e2 ok")

	// create competing candidates, with e1 initially losing to e2
	var wg sync.WaitGroup
	wg.Add(2)
	electc := make(chan *concurrency.Election, 2)
	//启动两个goroutine，竞争主节点
	go func() {
		defer wg.Done()
		// delay candidacy so e2 wins first
		time.Sleep(3 * time.Second)
		//节点名"e1"
		if err := e1.Campaign(context.Background(), "192.168.1.1"); err != nil {
			log.Fatal(err)
		}
		electc <- e1
	}()
	go func() {
		defer wg.Done()
		if err := e2.Campaign(context.Background(), "192.168.2.2"); err != nil {
			log.Fatal(err)
		}
		electc <- e2
	}()

	cctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	// electc 返回说明有主选出来，没选出来的goroutine会一直阻塞
	e := <-electc
	fmt.Println("completed first election with", string((<-e.Observe(cctx)).Kvs[0].Value))

	// 当前的主节点主动离职
	if err := e.Resign(context.TODO()); err != nil {
		log.Fatal(err)
	}
	//又有新节点当选主主节点
	e = <-electc
	fmt.Println("completed second election with", string((<-e.Observe(cctx)).Kvs[0].Value))

	wg.Wait()
	fmt.Println("over.........")
	time.Sleep(time.Hour)
	// Output:
	// completed first election with e2
	// completed second election with e1
}
