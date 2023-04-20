package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jursonmo/practise/pkg/proto"
	"github.com/jursonmo/practise/pkg/udp"
)

var addr = "0.0.0.0:9002"

func main() {
	server()
}

func server() {
	l, err := udp.NewUdpListen(context.Background(), "udp", addr, udp.WithReuseport(true), udp.WithListenerNum(2))
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second * 2) //让客户端先发出数据

	log.Printf("start accepting .....")

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) error {
	//echo
	msgHandler := func(pc *proto.ProtoConn, d []byte) error {
		fmt.Printf("receive from %v msg:%s\n", pc, string(d))
		_, err := pc.Write(d)
		if err != nil {
			log.Printf("server write back err:%v", err)
			return err
		}
		return nil
	}
	//设置proto.WithAuthHandler后，默认authOk == false, 即不允许收发用户数据， 表示需要验证通过后才行收发用户数据
	pconn := proto.NewProtoConn(conn, true, msgHandler,
		proto.WithHandShakeData(genHandshakeData), proto.WithAuthHandler(auth))

	ctx := context.Background()
	err := pconn.Handshake(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	//auth:
	err = pconn.Auth(ctx)
	if err != nil {
		log.Panic(err)
		return err
	}

	// udp server conn SetDeadline
	SetReadDeadline(pconn, true, time.Second*3)

	// run: read loop
	err = pconn.Run(ctx)
	if err != nil {
		log.Printf("Run err:%v\n", err)
	}
	//check write operation
	//超时后，测试下发送数据
	_, err = pconn.Write([]byte("xxxxx"))
	if err == nil {
		panic("expect write err")
	}
	log.Printf("after timeout, write err:%v\n", err)
	return err
}

//用于验证client 发过来的握手数据否跟服务器设定HandshakeData一致。
func genHandshakeData() []byte {
	return []byte("hello")
}

//输入的数据是client 发过来的auth request 数据，此处验证 auth request是否合法，然后回应是否auth OK， Ok 后才可以收发用户数据
func auth(d []byte) ([]byte, bool) {
	fmt.Printf("handler auth request data\n")
	req := AuthReq{}
	err := json.Unmarshal(d, &req)
	if err != nil {
		return []byte(err.Error()), false
	}
	if req.User != UserName {
		return []byte("user name auth fail"), false
	}
	if req.Pwd != Password {
		return []byte("password auth fail"), false
	}
	return []byte("we have conversation"), true
}
