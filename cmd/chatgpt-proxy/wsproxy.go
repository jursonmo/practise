package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/jursonmo/practise/pkg/dial"
)

func wsProxy(ctx context.Context, laddr, raddr string) error {
	chattingHandler := func(w http.ResponseWriter, r *http.Request) {
		wsconn, err := getWsConn(w, r)
		if err != nil {
			log.Printf("get ws conn err:%v", err)
			return
		}
		log.Printf("new ws conn:%v", info(wsconn.UnderlyingConn()))
		defer wsconn.Close()

		//connect server
		nctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		rconn, err := dial.Dial(nctx, raddr,
			dial.WithBackOffer(backoffx.NewDynamicBackoff(time.Second*1, time.Second*10, 1.5)),
			dial.WithKeepAlive(time.Second*20), dial.WithTcpUserTimeout(time.Second*5))
		if err != nil {
			wsconn.WriteMessage(websocket.BinaryMessage, []byte("proxy connect to server fail, try again a few minute later"))
			wsconn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		defer rconn.Close()

		go transReply(rconn, wsconn)

		//transfer to server
		for {
			_, message, err := wsconn.ReadMessage()
			if err != nil {
				log.Println("ws Error during message reading:", err)
				return
			}
			log.Printf("wsProxy Received from client: %s\n", string(message))

			if message[len(message)-1] != '\n' {
				message = append(message, '\n')
			}
			//log.Printf("%s, last byte:%v\n", message, message[len(message)-1])
			_, err = rconn.Write(message)
			if err != nil {
				log.Printf("write to server err:%v\n", err)
				wsconn.WriteMessage(websocket.BinaryMessage, []byte("proxy connect to server fail, try again later"))
				return
			}
		}
	}

	urls, err := url.Parse(laddr)
	if err != nil {
		return err
	}
	http.HandleFunc("/chatting", chattingHandler)
	http.HandleFunc("/", chatHome)
	log.Printf("ws listen on:%s\n", urls.Host)
	log.Fatal(http.ListenAndServe(urls.Host, nil))
	return nil
}

//websocket client 只接受payload
func transReply(rconn net.Conn, wsconn *websocket.Conn) error {
	defer rconn.Close()
	defer wsconn.Close()

	maxSize := 1024 * 32
	data := make([]byte, maxSize)
	for {
		_, err := io.ReadFull(rconn, data[:HeaderSize])
		if err != nil {
			log.Printf("read msg from server err:%v\n", err)
			return err
		}
		len := binary.BigEndian.Uint16(data[:HeaderSize])
		if int(len) > maxSize {
			log.Printf("len:%d to bigger", len)
			return err
		}
		_, err = io.ReadFull(rconn, data[:len])
		if err != nil {
			log.Printf("read msg from server err:%v\n", err)
			return err
		}
		log.Printf("reply payload len:%d\n", len)
		err = wsconn.WriteMessage(websocket.TextMessage, data[:len]) //用BinaryMessage 浏览器显示不出来
		if err != nil {
			log.Printf("write to client err:%v\n", err)
			return err
		}
		log.Printf("write reply to ws client ok:%d\n", len)
	}
}

func chatHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "chat.html")
}

var upgrader = websocket.Upgrader{} // use default options
func getWsConn(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// Upgrade our raw HTTP connection to a websocket based one
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return nil, err
	}

	SetReadDeadline(conn, true, 5*time.Minute) // 在收到ping msg 时，就重置read deadline
	return conn, nil
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Index Page")
}

func SetReadDeadline(conn *websocket.Conn, isServer bool, readDeadline time.Duration) {
	wrapHanlder := func(handler func(string) error, readDeadline time.Duration) func(string) error {
		msg := "pong"
		if isServer {
			msg = "ping"
		}
		return func(s string) error {
			log.Printf("receive %s from :%v, and SetReadDeadline:%v", msg, conn.RemoteAddr(), readDeadline)
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
