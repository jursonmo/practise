package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jursonmo/practise/pkg/proto/server"
	"github.com/jursonmo/practise/pkg/proto/session"
)

func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	return signals
}
func main() {
	srv, err := server.NewServer([]string{"tcp://0.0.0.0:9527"})
	if err != nil {
		panic(err)
	}
	//注册消息回调, msgid 必须大于10
	err = srv.AddRouter(11, session.HandleFunc(msg11Handle))
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	err = srv.Start(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("receive signal:%v\n", <-QuitSignal())
	srv.Stop(ctx)

	time.Sleep(time.Second)
	log.Println("over")
}

func msg11Handle(s session.Sessioner, msgid uint16, d []byte) {
	conn := s.UnderlayConn()
	log.Printf("session:%v->%v, receive msgid:%d, msg:%v", conn.LocalAddr(), conn.RemoteAddr(), msgid, string(d))

	//send msgid=12 msg
	err := s.WriteMsg(12, []byte("msg12"))
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("send msg12 ok")
}
