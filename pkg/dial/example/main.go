package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/jursonmo/practise/pkg/dial"
)

func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	return signals
}

//GOOS=linux go build main.go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//不设置keepalive, 默认idle, intvl 都是15秒, 所以这里设置成20，抓包验证服务器间隔多少秒发送keepalive
	s, err := dial.NewServer([]string{"tcp://0.0.0.0:8080", "tcp://0.0.0.0:8090"}, dial.ServerKeepalive(time.Second*20),
		dial.ServerUserTimeout(time.Second*5), dial.WithHandler(serverHandle))
	if err != nil {
		log.Panic(err)
	}
	go s.Start(ctx)

	//client start
	time.Sleep(time.Second)
	conn, err := dial.Dial(ctx, "tcp://127.0.0.1:8080",
		dial.WithBackOffer(backoffx.NewDynamicBackoff(time.Second*2, time.Second*30, 2.0)),
		dial.WithKeepAlive(time.Second*5), dial.WithTcpUserTimeout(time.Second*5))
	// conn, err := dial.Dial(ctx, "tcp://127.0.0.1:8080",
	// 	dial.WithTcpUserTimeout(time.Second*5))
	if err != nil {
		log.Panic(err)
	}

	//替换前面设置的5秒的idle intvl, 改成10 idle, 3秒intvl
	err = dial.SetKeepaliveParameters(conn, 10, 3, 3) //client 间隔10就发一次keepalive
	if err != nil {
		log.Println("SetKeepaliveParameters:", err)
	}

	data := []byte("123456\n")
	t := time.NewTimer(time.Second)
	quit := QuitSignal()
loop:
	for {
		select {
		case signal := <-quit:
			log.Printf("receive quit signal:%v\n", signal)
			cancel()
		case <-ctx.Done():
			log.Printf("ctx done,err:%v\n", ctx.Err())
			break loop
		case <-t.C:
			log.Printf("client write %s", string(data))
			_, err = conn.Write(data)
			if err != nil {
				return
			}
			t.Reset(time.Second * 40)
			//time.Sleep(time.Second * 40) //抓包查看 SetKeepaliveParameters 结果, 客户端10秒发一次keepalive, 服务器是20秒发一次
		}
	}
	s.Stop()
	time.Sleep(time.Second)
}

func serverHandle(conn net.Conn, fromLnID int) error {
	buf := make([]byte, 20)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Printf("server ln id:%d recv data:%s", fromLnID, string(buf[:n]))
	}
}
