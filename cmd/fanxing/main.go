package main

import (
	"encoding/json"
	"fmt"
)

type Person struct {
	Name string
}

func main() {
	p := &Person{"tom"}
	fanxingMarshal(p)
}

func fanxingMarshal[T any](v *T) {
	d, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(d))

	persions := []Person{}
	err = fanxingUnmarshal(&persions)
	fmt.Printf("persions:%v, err:%v\n", persions, err)
}

func fanxingUnmarshal[T any](v *[]T) error {
	data := []byte(`[
			{
				"name":"tom"
			},
			{
				"name":"tom2"
			}
		]`)

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}
