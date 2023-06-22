// client.go
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
	"golang.org/x/sys/unix"
)

//https://github.com/guybrand/WssSample/blob/master/goClient/goClient.go
var done chan interface{}
var interrupt chan os.Signal

//conn.ReadMessage() 只能读到应用数据，读不到控制信息的，比如ping/pong、close 等控制信息是读不到的。
func receiveHandler(connection *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		//connection.SetReadDeadline(time.Now().Add(5 * time.Second)) //每次read msg 后，即设置一次
		t, msg, err := connection.ReadMessage()
		if err != nil {
			log.Println("Error in receive:", err)
			//1. SetReadDeadline 超时后，这里打印:i/o timeout
			// Error in receive: read tcp 127.0.0.1:46852->127.0.0.1:8080: i/o timeout
			//2. tcp user timeout, 这里打印: connection timed out
			// Error in receive: read tcp 127.0.0.1:48046->127.0.0.1:8080: read: connection timed out
			//3. SetPongHandler 自定义接受pong信息时handler, 比如每次收到pong 回应时，就重置setReadDeadline()，超时这里打印：
			// Error in receive: read tcp 127.0.0.1:46852->127.0.0.1:8080: i/o timeout
			return
		}
		log.Printf("msg type:%v, Received: %s from:%v\n", t, msg, connection.RemoteAddr())
	}
}

//如果底层网络中途不通了，client 会一直发数据，感知不到底层网络的状况，怎么判断底层网络不通了呢
//1. 定期发送 自定义ping TextMessage, 不需要控制信息ping; receiveHandler 设置setReadDeadline()
//2. 定期发送控制信息ping; 底层conn, 设置tcp user timeout(发送数据后，规定的时间内没有收到ack,就超时，前提是有在发送数据，
//		所以, 如果想快速知道底层网络是否通，不能只设置tcp user timeout，还需要发送心跳的数据)
//3. 最好的方法是： 定期发送控制信息ping， wrap conn PongHander, 当接受到pong 回应时，就重置setReadDeadline()，
//	 如果一直收不到pong, receiveHandler 就会返回超时错误。
func main() {
	flag.Parse()
	// Channel to indicate that the receiverHandler is done
	interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT
	socketUrl := ""
	if len(os.Args) > 1 {
		socketUrl = "ws://" + os.Args[1] + "/socket"
	} else {
		socketUrl = "ws://localhost:8080" + "/socket"
	}
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wsDial(ctx, socketUrl)
		}()
		time.Sleep(time.Second * 2)
	}
	<-interrupt
	cancel()
	wg.Wait()
	log.Println("main over")
}

func wsDial(ctx context.Context, socketUrl string) {
	done := make(chan struct{})
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}
	defer conn.Close()

	go receiveHandler(conn, done)

	// Our main loop for the client
	// We send our relevant packets here
	pingTimer := time.NewTicker(time.Second)
	defer pingTimer.Stop()
	//tcp user timeout
	// if err := SetTcpUsertimeout(conn, 3*time.Second); err != nil {
	// 	log.Panic(err)
	// }

	SetReadDeadline(conn, false, 3*time.Second)

	sendTimer := time.NewTicker(time.Second * 3)
	defer sendTimer.Stop()
	for {
		select {
		case <-done:
			log.Println("Receiver Channel Closed! Exiting....")
			return
		case <-pingTimer.C:
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.PingMessage, []byte("ping msg"))
			if err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}
			fmt.Println("send PingMessage")
		case <-sendTimer.C:
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, []byte("Hello from GolangDocs!"))
			if err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}
			fmt.Println("send Hello from GolangDocs!")
		case <-ctx.Done():
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")

			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "")) //websocket.CloseNormalClosure
			if err != nil {
				log.Println("Error during closing websocket:", err)
				return
			}
			log.Println("send Close msg to server")
			select {
			case <-done:
				log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				//正常情况下，发送close 控制信息后，server 会关闭连接，本地收到fin 后, 当前的receiveHandler 会读取错误
				//本端发送close 控制信息并不会自己主动关闭本地socket
				//并且close(done), 上面的case <-done:会更早执行,这里就不会被调用到.
				//如果底层网络是不通的，receiveHandler 感知不到错误，那么就会走到这里。
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}

func SetTcpUsertimeout(conn *websocket.Conn, t time.Duration) error {
	underlyConn := conn.UnderlyingConn()
	tconn := ConvertTcpConn(underlyConn)
	if tconn == nil {
		return fmt.Errorf("get underlay tcp conn fail")
	}
	rawConn, err := tconn.SyscallConn()
	if err != nil {
		log.Printf("on getting raw connection object for keepalive parameter setting", err.Error())
		return err
	}

	rawConn.Control(
		func(fdPtr uintptr) {
			fd := int(fdPtr)
			err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, int(t.Milliseconds()))
		})
	return err
}

func ConvertTcpConn(conn net.Conn) *net.TCPConn {
	if conn == nil {
		return nil
	}
	if !strings.Contains(conn.LocalAddr().Network(), "tcp") /* && !strings.Contains(conn.LocalAddr().Network(), "tls")*/ {
		log.Printf("unsupport conn network:%s \n", conn.LocalAddr().Network())
		return nil
	}
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		return tcpconn
	}

	tlsconn, ok2 := conn.(*tls.Conn)
	if !ok2 {
		return nil
	}

	type tc struct {
		underConn net.Conn
	}
	tconn := (*tc)(unsafe.Pointer(tlsconn))
	tcpconn, ok = tconn.underConn.(*net.TCPConn)
	if !ok {
		return nil
	}
	return tcpconn
}

func SetReadDeadline(conn *websocket.Conn, isServer bool, readDeadline time.Duration) {
	wrapHanlder := func(handler func(string) error, readDeadline time.Duration) func(string) error {
		msg := "pong"
		if isServer {
			msg = "ping"
		}
		return func(s string) error {
			log.Printf("receive %s from :%v, and SetReadDeadline:%v", msg, conn.RemoteAddr(), readDeadline)
			//每次收到报文都重置ReadDeadline，如果超过readDeadline 没有收到指定数据，即没有走到这里，就会超时
			//虽然可以在收到任意报文时重置ReadDeadline，但是这样在大流量的情况下会频繁设置SetReadDeadline，有一定的性能损耗
			conn.SetReadDeadline(time.Now().Add(readDeadline))
			if handler != nil {
				return handler(s)
			}
			return nil
		}
	}

	var handler func(string) error
	if isServer {
		handler = conn.PingHandler()
		conn.SetPingHandler(wrapHanlder(handler, readDeadline))
	} else {
		handler = conn.PongHandler()
		conn.SetPongHandler(wrapHanlder(handler, readDeadline))
	}
}

//just for client conn to SetReadDeadline
func ClientSetReadDeadline(conn *websocket.Conn, readDeadline time.Duration) {
	handler := conn.PongHandler()
	wrapHandler := func(s string) error {
		log.Printf("client receive pong from :%v, and SetReadDeadline :%v", conn.RemoteAddr(), readDeadline)
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		if handler != nil {
			return handler(s)
		}
		return nil
	}
	log.Println("SetPongHandler with read deadline")
	conn.SetPongHandler(wrapHandler)
}
