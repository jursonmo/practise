package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

func main() {
	var d int
	var err error
	if len(os.Args) > 1 {
		d, err = strconv.Atoi(os.Args[1])
		if err != nil {
			panic(err)
		}
	} else {
		d = 60
	}
	// 创建一个redis的客户端连接
	client := goredislib.NewClient(&goredislib.Options{
		Addr: "localhost:6379",
	})
	// 创建redsync的客户端连接池
	pool := goredis.NewPool(client) // or, pool := redigo.NewPool(...)

	// 创建redsync实例
	rs := redsync.New(pool)

	// 通过相同的key值名获取同一个互斥锁.
	mutexname := "my-global-mutex"
	//创建基于key的互斥锁
	mutex := rs.NewMutex(mutexname)

	fmt.Printf("start get lock\n")
	// 对key进行
	if err := mutex.Lock(); err != nil {
		panic(err)
	}

	// 获取锁后的业务逻辑处理.
	locktime := time.Second * time.Duration(d)
	fmt.Printf("get locked, for %v\n", locktime)
	time.Sleep(locktime)
	// 释放互斥锁
	if ok, err := mutex.Unlock(); !ok || err != nil {
		panic(fmt.Sprintf("unlock failed, err:%v", err))
	}
	fmt.Println("----------overr -----------")
}
