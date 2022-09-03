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
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/nosleep",
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
底层只有一个tcp, 因为每次请求，都立即得到response,同时也被我们读出来，所以这个底层tcp 连接是可以服用的。
MacBook-Pro:~ obc$ netstat -anl|grep 1313
tcp46      0      0  *.1313                 *.*                    LISTEN
tcp4       0      0  127.0.0.1.53529        127.0.0.1.1313         TIME_WAIT
*/
