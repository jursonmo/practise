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
	ra, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, ra)
	if err != nil {
		panic(err)
	}

	pconn := proto.NewProtoConn(conn, false, msgHandler,
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

	//设置超时, 多久没有收到pong 回应超时
	SetReadDeadline(pconn, false, time.Second*5)

	//run: for reading data loop
	go pconn.Run(ctx)

	//write data
	i := 0
	wTick := time.NewTicker(time.Second)
	defer wTick.Stop()

	pingTick := time.NewTicker(time.Second * 2)
	defer pingTick.Stop()

loop:
	for {
		select {
		case <-wTick.C:
			i++
			msg := fmt.Sprintf("%d", i)
			fmt.Printf("write %s \n", msg)
			_, err := pconn.Write([]byte(msg))
			if err != nil {
				log.Println(err)
				return
			}
			if i > 5 {
				log.Printf("退出发送数据\n")
				pingTick.Stop()
				break loop
			}
		case <-pingTick.C:
			err := pconn.WritePing([]byte("ping"))
			if err != nil {
				log.Printf("write ping err:%v", err)
				return
			}
		}
	}

	//客户端没有再发ping消息, 等待client 主动超时
	log.Printf("等待客户端超时。。。。\n")
	time.Sleep(time.Hour)
}

func msgHandler(pc *proto.ProtoConn, d []byte) error {
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
