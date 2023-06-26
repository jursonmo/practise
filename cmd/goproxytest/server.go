package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {

	// http.HandleFunc("/echo", echo)
	// http.HandleFunc("/", http.NotFound)
	//err := http.ListenAndServe(":1313", nil)
	//fmt.Println(err)

	mux := http.NewServeMux()
	mux.HandleFunc("/echo", echo)
	http.HandleFunc("/", http.NotFound)
	go func() {
		fmt.Printf("http listen on 1313\n")
		err := http.ListenAndServe(":1313", mux)
		fmt.Println(err)
	}()

	go func() {
		fmt.Printf("https listen on 1443\n")
		err := http.ListenAndServeTLS(":1443", "server.pem", "server.key", mux)
		fmt.Println("tls:", err)
	}()

	time.Sleep(time.Hour * 10)
	return
}

func echo(w http.ResponseWriter, req *http.Request) {
	//正常接收get.go 请求：echo handler, method:GET, Host:127.0.0.1:1313, url:/echo
	//接收代理tr.Proxy的请求：echo handler, method:GET, Host:127.0.0.2:1313, url:http://127.0.0.2:1313/echo
	//CONNECT 的请求，这里HOST 可以是任意，不一定是127.0.0.1:1313, 比如打印echo handler, method:Get, Host:1.1.1.1:1234, url:/echo
	fmt.Printf("echo handler, method:%s, Host:%v, url:%#v\n", req.Method, req.Host, req.URL)
	w.Write([]byte("echo"))
}
