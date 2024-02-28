package main

import (
	"fmt"
	"time"

	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/znet"
)

// User-defined heartbeat message processing method
// 用户自定义的心跳检测消息处理方法
func myHeartBeatMsg(conn ziface.IConnection) []byte {
	return []byte("heartbeat, I am server, I am alive")
}

// User-defined handling method for remote connection not alive.
// 用户自定义的远程连接不存活时的处理方法
func myOnRemoteNotAlive(conn ziface.IConnection) {
	fmt.Println("myOnRemoteNotAlive is Called, connID=", conn.GetConnID(), "remoteAddr = ", conn.RemoteAddr())
	//关闭链接
	conn.Stop()
}

// User-defined method for handling heartbeat messages (用户自定义的心跳检测消息处理方法)
type myHeartBeatRouter struct {
	znet.BaseRouter
}

func (r *myHeartBeatRouter) Handle(request ziface.IRequest) {
	fmt.Println("in MyHeartBeatRouter Handle, recv from client : msgId=", request.GetMsgID(), ", data=", string(request.GetData()))
}

func main() {
	s := znet.NewServer()

	myHeartBeatMsgID := 88888

	// Start heartbeating detection. (启动心跳检测)
	s.StartHeartBeatWithOption(1*time.Second, &ziface.HeartBeatOption{
		MakeMsg:          myHeartBeatMsg,
		OnRemoteNotAlive: myOnRemoteNotAlive,
		Router:           &myHeartBeatRouter{},
		HeartBeatMsgID:   uint32(myHeartBeatMsgID),
	})
	//服务启动后， 会启动 服务的心跳机制, 定时去检测连接是否IsAlive(), 如何不是Alive(), 就调用回调onRemoteNotAlive 通知业务层
	//，连接收到一个报文，就会更新lastActivityTime
	s.Serve()
}

/*
func (h *HeartbeatChecker) check() (err error) {

	if h.conn == nil {
		return nil
	}
 	//莫：定时去检测连接是否存活
	if !h.conn.IsAlive() {
		h.onRemoteNotAlive(h.conn)
	} else {
		if h.beatFunc != nil {
			err = h.beatFunc(h.conn)
		} else {
			err = h.SendHeartBeatMsg()
		}
	}

	return err
}
//判断 检测连接是否存活，其实就是判断c.lastActivityTime 最后收到报文的时间是否超时。
func (c *Connection) IsAlive() bool {
	if c.isClosed {
		return false
	}
	// Check the last activity time of the connection. If it's beyond the heartbeat interval,
	// then the connection is considered dead.
	// (检查连接最后一次活动时间，如果超过心跳间隔，则认为连接已经死亡)
	return time.Now().Sub(c.lastActivityTime) < zconf.GlobalObject.HeartbeatMaxDuration()
}
*/
