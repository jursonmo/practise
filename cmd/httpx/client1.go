package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

//强行设置每个host 底层只有一个tcp, 测试下结果,
//服务器也是先处理完最先到达的请求后，再处理下一个，即使下一个请求其实很早就到达服务器了
func main() {
	client := &http.Client{}
	transport := http.DefaultTransport.(*http.Transport)
	transport.MaxConnsPerHost = 1 //强行设置每个host 底层只有一个tcp,
	client.Transport = transport
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
		//最后发送，最后得到response
		i := 2
		log.Printf("i:%d start \n", i)
		request(client, req)
		log.Printf("i:%d over\n", i)
	}()
	go func() {
		defer wg.Done()
		//最先发送，最先得到response
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
		log.Println(err)
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(data))
}

/*
51334 <--> 1313 端口，他们是同一个tcp 连接

MacBook-Pro:httpx obc$ netstat -anl|grep 1313
tcp4       0      0  127.0.0.1.1313         127.0.0.1.51334        ESTABLISHED
tcp4       0      0  127.0.0.1.51334        127.0.0.1.1313         ESTABLISHED
tcp46      0      0  *.1313
*/
