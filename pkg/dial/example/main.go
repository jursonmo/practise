package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jursonmo/practise/pkg/dial"
)

//GOOS=linux go build main.go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := dial.NewServer("tcp://0.0.0.0:8080", dial.ServerKeepalive(time.Second*5),
		dial.ServerUserTimeout(time.Second*5), dial.WithHandler(serverHandle))
	if err != nil {
		panic(err)
	}
	go s.Start(ctx)

	time.Sleep(time.Second)
	conn, err := dial.Dial(ctx, "tcp://127.0.0.1:8080",
		dial.WithKeepAlive(time.Second*5), dial.WithTcpUserTimeout(time.Second*5))
	if err != nil {
		panic(err)
	}
	for {
		time.Sleep(time.Second)
		_, err = conn.Write([]byte("123456\n"))
		if err != nil {
			return
		}
	}
}

func serverHandle(conn net.Conn) error {
	buf := make([]byte, 20)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Printf("data:%s", string(buf[:n]))
	}
}
