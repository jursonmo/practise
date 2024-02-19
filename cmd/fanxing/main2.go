package main

import (
	"encoding/json"
	"fmt"
)

type API struct {
	MyTarget string `json:"target"`
}

type API2 struct {
	MyTarget2 string `json:"target2"`
}

type API3 struct {
	MyTarget3 string `json:"target3"`
}

type ApiType interface {
	API | API2 | API3
}

func decode[T ApiType](s string, t *T) {
	err := json.Unmarshal([]byte(s), t)
	if err != nil {
		fmt.Println("解析异常")
	}
}

func main() {
	s1 := "{\"target\":\"target\"}"
	api1 := new(API)
	decode(s1, api1)
	fmt.Printf("%+v\n", api1)

	s2 := "{\"target2\":\"target2\"}"
	api2 := new(API2)
	decode(s2, api2)
	fmt.Printf("%+v\n", api2)

	s3 := "{\"target3\":\"target3\"}"
	api3 := new(API3)
	decode(s3, api3)
	fmt.Printf("%+v\n", api3)
}
