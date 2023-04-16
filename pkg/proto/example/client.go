package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jursonmo/practise/pkg/proto"
)

var addr = "127.0.0.1:9001"

type Person struct {
	Name string
	Age  int
}

func main() {
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	// fmt.Printf("1 <<1 :%d\n", 1<<1)
	// fmt.Printf("1<<2 :%d\n", 1<<2)
	// fmt.Printf("1<<3:%d\n", 1<<3)
	// fmt.Printf("1<<4, 一向左移4 :%d\n", 1<<4)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}

	r := bufio.NewReader(conn)
	for {
		//1. -------------------
		p := Person{"mjw", 30}
		pkg := proto.NewProtoPkg()
		pkg.Marshal(p, "json")
		fmt.Printf("write pkg :%v\n", pkg)
		_, err := conn.Write(pkg.Bytes())
		if err != nil {
			fmt.Println("Write", err)
			os.Exit(1)
		}
		pkg1 := proto.NewProtoPkg()
		err = pkg1.Decode(r)
		if err != nil {
			fmt.Println("Decode", err)
			os.Exit(1)
		}
		p1 := Person{}
		err = pkg1.Unmarshal(&p1)
		if err != nil {
			fmt.Println("Unmarshal", err)
			os.Exit(1)
		}
		fmt.Printf("reply, person:%v\n", p1)
		time.Sleep(time.Second * 2)

		//2.-----------------------
		pingPkg, err := proto.NewPingPkg(nil)
		if err != nil {
			fmt.Println("NewPingPkg", err)
			os.Exit(1)
		}
		_, err = conn.Write(pingPkg.Bytes())
		if err != nil {
			fmt.Println("Write", err)
			os.Exit(1)
		}
		pongPkg := proto.NewProtoPkg()
		err = pongPkg.Decode(r)
		if err != nil {
			fmt.Println("Decode", err)
			os.Exit(1)
		}
		fmt.Printf("pongPkg:%v\n", pongPkg)
		time.Sleep(time.Second * 2)

		//3.-------------------------------
	}
}
