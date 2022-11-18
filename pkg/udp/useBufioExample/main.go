package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/jursonmo/practise/pkg/udp"
)

func main() {
	server()
	time.Sleep(time.Millisecond * 10)
	client()
}

func server() {
	l, err := udp.NewUdpListen(context.Background(), "udp", "0.0.0.0:3333",
		udp.WithReuseport(true), udp.WithListenerNum(2))
	if err != nil {
		panic(err)
	}

	go func() {
		log.Printf("start accepting .....")
		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}
			go handle(conn)
		}
	}()
}

func handle(conn net.Conn) {
	log.Printf("accpet new conn:%v--%v", conn.LocalAddr(), conn.RemoteAddr())
	ch := make(chan []byte, 10)

	go func() {
		bw := udp.NewBufioWriter(conn, 8)
		time.Sleep(time.Second) //等ch 里有一定的数据，这样测试 bufio -->flush -->WriteBatch
		for data := range ch {
			_, err := bw.Write(data) //测试通过bufio 来WriteBatch,减少系统调用
			if err != nil {
				log.Println(err)
				return
			}
			if len(ch) == 0 && bw.Buffered() > 0 {
				err = bw.Flush()
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}()

	//对于accept 产生的net.Conn
	//目前不能通过bufio 达到手动调用readBatch 以减少系统调用的目的,
	//这里看似一个一个Read, 实际并也不是每次read 都发生一次系统调用
	//因为默认后台已经有个任务在listen socket 上readBatch 了。
	//(listen socket是任意client packet 进来的接口, 是判断是否产生新UDPConn 关键）
	//但可以手动调用batch write 到达减少系统调用
	br := udp.NewBufioReader(conn, 8, 1024)
	if br == nil {
		log.Printf("NewBufioReader return nil")
		br = conn
	}
	buf := make([]byte, 1600)
	i := 0
	for {
		i++
		n, err := br.Read(buf)
		if err != nil {
			panic(err)
		}

		log.Printf("server %d recv data:%s", i, string(buf[:n]))
		ch <- buf[:n]
	}

}

func client() {
	//udp.WithRxHandler(nil) 表示不需要后台起一个goroutine 来负责读数据，由我自己手动调用read or readBatch
	//否则后台起一个goroutine 来负责读数据, conn.Read() 就可以读到数据，原应用层的代码就不需要改动
	conn, err := udp.UdpDial(context.Background(), "udp", "", "127.0.0.1:3333", udp.WithRxHandler(nil))
	if err != nil {
		panic(err)
	}
	log.Printf("dial ok, %v--%v", conn.LocalAddr(), conn.RemoteAddr())
	go func() {
		br := udp.NewBufioReader(conn, 8, 1600)
		buf := make([]byte, 2048)
		i := 0
		time.Sleep(time.Second * 2)
		for {
			i++
			n, err := br.Read(buf)
			if err != nil {
				panic(err)
			}
			log.Printf("client read i:%d,  data:%s", i, string(buf[:n]))
		}
	}()

	ch := make(chan []byte, 10)
	data := []byte("12345678")
	for i := 0; i < 10; i++ {
		ch <- data
	}

	bw := udp.NewBufioWriter(conn, 8)
	for data := range ch {
		_, err = bw.Write(data)
		if err != nil {
			log.Println(err)
			return
		}
		if len(ch) == 0 && bw.Buffered() > 0 {
			err = bw.Flush()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}

}

/*
root@ubuntu:~/udp# ./main
2022/11/19 01:17:28 start accepting .....
2022/11/19 01:17:28 listener, id:0, local:[::]:3333 listenning....
2022/11/19 01:17:28 listener, id:1, local:[::]:3333 listenning....
2022/11/19 01:17:28 dial ok, 127.0.0.1:51767--127.0.0.1:3333
2022/11/19 01:17:28 127.0.0.1:51767->127.0.0.1:3333, flushing 8 packet....
2022/11/19 01:17:28 127.0.0.1:51767->127.0.0.1:3333, flushing 2 packet....
2022/11/19 01:17:28 id:0, got n:8, len(ms):8
2022/11/19 01:17:28 listener, id:0, local:[::]:3333, new conn:127.0.0.1:51767
2022/11/19 01:17:28 id:0, got n:2, len(ms):8
2022/11/19 01:17:28 accpet new conn:[::]:3333--127.0.0.1:51767
2022/11/19 01:17:28 NewBufioReader return nil
2022/11/19 01:17:28 server 1 recv data:12345678
2022/11/19 01:17:28 server 2 recv data:12345678
2022/11/19 01:17:28 server 3 recv data:12345678
2022/11/19 01:17:28 server 4 recv data:12345678
2022/11/19 01:17:28 server 5 recv data:12345678
2022/11/19 01:17:28 server 6 recv data:12345678
2022/11/19 01:17:28 server 7 recv data:12345678
2022/11/19 01:17:28 server 8 recv data:12345678
2022/11/19 01:17:28 server 9 recv data:12345678
2022/11/19 01:17:28 server 10 recv data:12345678
2022/11/19 01:17:29 [::]:3333->127.0.0.1:51767, flushing 8 packet....
2022/11/19 01:17:29 [::]:3333->127.0.0.1:51767, flushing 2 packet....
2022/11/19 01:17:30 127.0.0.1:51767<-127.0.0.1:3333, batch got n:8, len(ms):8
2022/11/19 01:17:30 127.0.0.1:51767<-127.0.0.1:3333, batch got n:2, len(ms):8
*/
