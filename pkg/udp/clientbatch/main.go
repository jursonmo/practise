package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/jursonmo/practise/pkg/udp"
)

func main() {
	go server()
	time.Sleep(time.Second)

	data := []byte("12345678")
	client("udp", "127.0.0.1:3333", data)
}

func server() {
	l, err := udp.NewUdpListen(context.Background(), "udp", "0.0.0.0:3333", udp.WithReuseport(true), udp.WithListenerNum(2))
	if err != nil {
		panic(err)
	}
	//time.Sleep(time.Second * 2) //让客户端先发出数据

	log.Printf("start accepting .....")

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	buf := make([]byte, 1600)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		log.Printf("server recv data:%s, and write back", string(buf[:n]))

		wn, err := conn.Write(buf[:n])
		if err != nil {
			panic(err)
		}
		if n != wn {
			log.Panicf("n:%d, wn:%d", n, wn)
		}
	}
}

func client(network, raddr string, data []byte) {
	uconn, err := udp.UdpDial(context.Background(), "udp", "", "127.0.0.1:3333")
	if err != nil {
		log.Panic(err)
	}
	for i := 0; i < 10; i++ {
		n, err := uconn.Write(data)
		if err != nil {
			panic(err)
		}
		log.Printf("%d, conn local:%s->%s, write data len:%d", i, uconn.LocalAddr().String(),
			uconn.RemoteAddr().String(), n)
	}
	buf := make([]byte, 1600)
	i := 0
	for {
		n, err := uconn.Read(buf)
		if err != nil {
			panic(err)
		}
		log.Printf("i:%d, client recv data:%s", i, string(buf[:n]))
		i++
	}
}
