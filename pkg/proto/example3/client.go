package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/jursonmo/practise/pkg/proto"
)

//go build client.go auth.go
var addr = "127.0.0.1:9002"

type Person struct {
	Name string
	Age  int
}

func main() {
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	ctx := context.Background()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	pconn := proto.NewProtoConn(conn, false, proto.ProtoMsgHandle(msgHandler),
		proto.WithHandShakeData(genHandshakeData), proto.WithAuthReqData(genAuthReqData))
	//init:
	err = pconn.Handshake(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	//auth:
	err = pconn.Auth(ctx)
	if err != nil {
		log.Panicf("auth err:%v", err)
		return
	}
	//run:
	go pconn.Run(ctx)

	//write data
	i := 0
	for {
		i++
		if i > 3 {
			fmt.Printf("send close msg\n")
			_, err := pconn.WriteCloseMsg(proto.CloseNormalClosure, "client have no data to send")
			if err != nil {
				log.Panic(err)
			}
			time.Sleep(time.Second)
			return
		}
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
	fmt.Printf("receive from %v msg:%s\n", pc, string(d))
	return nil
}

//返回的数据就是握手时发送的数据
func genHandshakeData() []byte {
	return []byte("hello")
}

//返回的数据就是auth request时发送的数据
func genAuthReqData() []byte {
	d, _ := json.Marshal(&authReq)
	return d
}
