package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/sleep", handler2)
	http.HandleFunc("/nosleep", handler3)
	http.HandleFunc("/echo", echo)
	err := http.ListenAndServe(":1313", nil)
	fmt.Println(err)
	return
}

type reqData struct {
	Sleep string
}

var sleep = 20

func echo(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("echo handler, method:%s, url:%v\n", req.Method, req.URL)
	w.Write([]byte("echo"))
}

func handler2(w http.ResponseWriter, req *http.Request) {
	tmp := sleep
	log.Printf("sleep %d", tmp)
	sleep -= 10
	time.Sleep(time.Second * time.Duration(tmp))
	log.Printf("sleep %d over", tmp)
	w.Write([]byte(strconv.Itoa(tmp)))
}

var index int

func handler3(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(strconv.Itoa(index)))
	index++
}
func handler(w http.ResponseWriter, req *http.Request) {
	// req.ParseForm()
	// sleep := req.Form.Get("sleep")
	data, err := io.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	reqdata := &reqData{}
	err = json.Unmarshal(data, reqdata)
	if err != nil {
		panic(err)
	}
	fmt.Printf("get %v\n", reqdata)
	sleep := reqdata.Sleep
	s, err := strconv.Atoi(sleep)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second * time.Duration(s))
	w.Write([]byte(sleep))
	fmt.Printf("sleep :%d,response\n", s)
}
