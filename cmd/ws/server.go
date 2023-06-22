// server.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options
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

//just for server conn
func ServerSetReadDeadline(conn *websocket.Conn, readDeadline time.Duration) {
	handler := conn.PingHandler()
	wrapHandler := func(s string) error {
		log.Printf("server receive ping from :%v, and SetReadDeadline :%v", conn.RemoteAddr(), readDeadline)
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		if handler != nil {
			return handler(s)
		}
		return nil
	}
	log.Println("SetPingHandler with read deadline")
	conn.SetPingHandler(wrapHandler)
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer conn.Close()

	//conn.SetPingHandler()
	// underlyConn := conn.UnderlyingConn()
	// _ = underlyConn

	// The event loop
	//ServerSetReadDeadline(conn, 5*time.Second)
	SetReadDeadline(conn, true, 5*time.Second) // 在收到ping msg 时，就重置read deadline
	for {
		//conn.ReadMessage() 只能读到应用数据，读不到控制信息的，比如ping/pong、close 等控制信息是读不到的。
		//conn.SetReadDeadline(time.Now().Add(5 * time.Second)) //控制信息 ping msg 不能重置
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}
		log.Printf("Received: %s", message)
		//time.Sleep(time.Second * 6)
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Println("Error during message writing:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Index Page")
}

func main() {
	http.HandleFunc("/socket", socketHandler)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
