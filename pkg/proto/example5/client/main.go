package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jursonmo/practise/pkg/proto/client"
	"github.com/jursonmo/practise/pkg/proto/session"
)

func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	return signals
}

func main() {
	cli, err := client.NewClient(
		[]string{"tcp://127.0.0.1:9527"},
		client.WithOnConnect(connectHandle),
		client.WithOnDialFail(dialFail),
	)
	if err != nil {
		panic(err)
	}
	//注册消息回调
	err = cli.AddRouter(12, session.HandleFunc(msgHandle))
	if err != nil {
		panic(err)
	}

	//启动client
	ctx := context.Background()
	err = cli.Start(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("receive signal:%v\n", <-QuitSignal())
	cli.Stop(ctx)

	time.Sleep(time.Second)
	log.Println("over")
}

func dialFail(err error) {
	log.Printf("dial fail, err:%v", err)
}

func connectHandle(s session.Sessioner) error {

	for {
		//bug:如果client 断开后，重连比较快，那么Client{}.WriteMsg 不回返回错误，因为Client.conn 是新的正常的，也就是这个for循环一直可以发送，
		//重连成功后产生新的goroutine for也能一直发送,所以需要Sessioner 不能一直指向Client{} 这个一直存在的对象
		//也就是每次重连后，Sessioner 都指向新的对象， 旧的对象的底层conn 是断开的状态，这样旧的goroutine writeMsg就能感知错误并退出。
		err := s.WriteMsg(11, []byte("msg11"))
		if err != nil {
			log.Printf("WriteMsg err:%v", err)
			return err
		}
		log.Println("send id 11 msg ok on ", s.SessionID())
		time.Sleep(time.Second * 2)
	}

}

func msgHandle(s session.Sessioner, id uint16, d []byte) {
	conn := s.UnderlayConn()
	log.Printf("receive msgid:%d, msg:%v, session:%v->%v", id, string(d), conn.LocalAddr(), conn.RemoteAddr())
}
