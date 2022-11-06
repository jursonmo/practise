package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	//client 得到response 跟服务器的处理有关， 先回复，client 就先拿到response， 原因是底层tcp 不止一个
	// 结论是服务器先数据的顺序跟 client 请求的顺序不一样的
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/sleep",
		nil)
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		time.Sleep(time.Second)
		i := 1
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 2)
		//最后发送，由于服务器sleep 时间最短，依然能最先得到response
		i := 2
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	go func() {
		defer wg.Done()
		//最先发送，最后得到response
		i := 0
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	wg.Wait()

	req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/nosleep",
		nil)
	if err != nil {
		panic(err)
	}
	wg.Add(3)

	go func() {
		defer wg.Done()
		time.Sleep(time.Second)
		i := 1
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 2)
		i := 2
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	go func() {
		defer wg.Done()
		i := 0
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	wg.Wait()
}
func request(client *http.Client, req *http.Request) {
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

/*
看了底层是三个tcp 连接，怪不得，如果是一个tcp 连接，是做不到的
MacBook-Pro:~ obc$ netstat -anl|grep 1313
tcp46      0      0  *.1313                 *.*                    LISTEN
tcp4       0      0  127.0.0.1.52846        127.0.0.1.1313         TIME_WAIT
tcp4       0      0  127.0.0.1.52844        127.0.0.1.1313         TIME_WAIT
tcp4       0      0  127.0.0.1.52848        127.0.0.1.1313         TIME_WAIT

抓包看是http 1.1
http1 可以同时多个request 是因为底层发起多个tcp连接，
http2 已经是多路复用，一个tcp连接里多个http request,
http3 底层是udp quic
*/
