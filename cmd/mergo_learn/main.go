package main

import (
	"fmt"
	"log"

	"github.com/imdario/mergo"
)

type redisConfig struct {
	Address string
	Port    int
	DB      int
}

var defaultConfig = redisConfig{
	Address: "127.0.0.1",
	Port:    6381,
	DB:      1,
}

func main() {
	var config redisConfig
	config.DB = 2
	//参数 1 是目标对象，参数 2 是源对象，这两个函数的功能就是将源对象中的字段复制到目标对象的对应字段上。
	if err := mergo.Merge(&config, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println("redis address: ", config.Address)
	fmt.Println("redis port: ", config.Port)
	fmt.Println("redis db: ", config.DB)
	cc := defaultConfig
	cc.DB = 0

	if err := mergo.Merge(&cc, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println("cc redis address: ", cc.Address)
	fmt.Println("cc redis port: ", cc.Port)
	fmt.Println("cc redis db: ", cc.DB) // cc.DB= 1, 即如果源的字段是默认值,会被替换

	//struct to map
	var m = make(map[string]interface{})
	if err := mergo.Map(&m, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println(m)

	var ms = make(map[string]string)
	if err := mergo.Map(&ms, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println(ms)

	//map to Struct
	mapToStruct()
}

type Student struct {
	Name string
	Num  int
	Age  int
}

func mapToStruct() {
	var defaultStudent = Student{}

	var m = make(map[string]interface{})
	m["name"] = "lisi"
	m["num"] = 2
	m["age"] = 20

	if err := mergo.Map(&defaultStudent, m); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("struct defaultStudent = %+v", defaultStudent)
}
