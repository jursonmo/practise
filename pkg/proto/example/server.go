package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/jursonmo/practise/pkg/proto"
)

var port = "0.0.0.0:9001"

type Person struct {
	Name string
	Age  int
}

func echo(conn net.Conn) {
	r := bufio.NewReader(conn)
	for {
		pkg := proto.NewProtoPkg()
		err := pkg.Decode(r)
		if err != nil {
			fmt.Println("Decode", err)
			os.Exit(1)
		}
		p1 := Person{}
		err = pkg.Unmarshal(&p1)
		if err != nil {
			fmt.Println("Unmarshal", err)
			fmt.Printf("receive pkg:%v\n", pkg)
		} else {
			fmt.Printf("receive, person:%v\n", p1)
		}

		_, err = conn.Write(pkg.Bytes())
		if err != nil {
			fmt.Println("Write", err)
			os.Exit(1)
		}
	}
}

func main() {
	fmt.Printf("listenning at %s\n", port)
	l, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("ERROR", err)
			continue
		}
		go echo(conn)
	}
}
