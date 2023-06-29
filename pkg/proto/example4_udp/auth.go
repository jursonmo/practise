package main

import (
	"log"
	"time"

	"github.com/jursonmo/practise/pkg/proto"
)

type AuthReq struct {
	User string
	Pwd  string
}

var UserName = "tom"
var Password = "123456"
var authReq = AuthReq{User: UserName, Pwd: Password}

func SetReadDeadline(conn *proto.ProtoConn, isServer bool, readDeadline time.Duration) {
	//fix:先设置一个deadline, 如果没有收到ping or pong 自动超时, 收到ping or pong 后 会更新deadline
	conn.SetReadDeadline(time.Now().Add(readDeadline)) //如果这里不设置deadline，可能一直收不到ping,那么永远都无法超时

	wrapHanlder := func(handler func([]byte) error, readDeadline time.Duration) func([]byte) error {
		msg := "pong"
		if isServer {
			msg = "ping"
		}
		return func(d []byte) error {
			log.Printf("receive %s from :%v, and SetReadDeadline:%v", msg, conn.RemoteAddr(), readDeadline)
			//每次收到指定报文都重置ReadDeadline，如果超过readDeadline 没有收到指定数据(ping or pong)，即没有走到这里，就会超时
			//虽然可以在收到任意报文时重置ReadDeadline，但是这样在大流量的情况下会频繁设置SetReadDeadline，有一定的性能损耗
			conn.SetReadDeadline(time.Now().Add(readDeadline))
			if handler != nil {
				return handler(d)
			}
			return nil
		}
	}

	var handler func([]byte) error
	if isServer {
		handler = conn.PingHandler()
		conn.SetPingHandler(wrapHanlder(handler, readDeadline))
	} else {
		handler = conn.PongHandler()
		conn.SetPongHandler(wrapHanlder(handler, readDeadline))
	}
}
