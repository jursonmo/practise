package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/jursonmo/practise/pkg/proto"
)

var port = "0.0.0.0:9002"

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
		msgHandler := func(pc *proto.ProtoConn, d []byte) error {
			fmt.Printf("receive msg:%s\n", string(d))
			return nil
		}
		pconn := proto.NewProtoConn(conn, true, msgHandler, proto.WithHandShakeData(genHandshakeData))
		go func() {
			err = pconn.Handshake(context.Background())
			if err != nil {
				log.Println(err)
				return
			}
			err = pconn.Run(context.Background())
			if err != nil {
				log.Printf("Run err:%v\n", err)
				return
			}
		}()
	}
}

//用于验证client 发过来的握手数据否跟服务器设定HandshakeData一致。
func genHandshakeData() []byte {
	return []byte("hello")
}
