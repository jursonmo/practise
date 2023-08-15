package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/jursonmo/practise/pkg/proto"
)

var addr = "127.0.0.1:9002"

type Person struct {
	Name string
	Age  int
}

func main() {
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}

	pconn := proto.NewProtoConn(conn, false, proto.ProtoMsgHandle(msgHandler), proto.WithHandShakeData(genHandshakeData))
	//init:
	err = pconn.Handshake(context.Background())
	if err != nil {
		log.Panic(err)
		return
	}
	//run:
	go pconn.Run(context.Background())

	//write data
	i := 0
	for {
		i++
		msg := fmt.Sprintf("%d", i)
		fmt.Printf("write %s \n", msg)
		_, err := pconn.Write([]byte(msg))
		if err != nil {
			log.Println(err)
			return
		}
		time.Sleep(time.Second * 2)
	}
}

func msgHandler(pc *proto.ProtoConn, d []byte, t byte) error {
	fmt.Printf("receive msg:%s", string(d))
	return nil
}

//返回的数据就是握手时发送的数据
func genHandshakeData() []byte {
	return []byte("hello")
}
