package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"

	//"github.com/brucespang/go-tcpinfo"
	"github.com/jursonmo/go-tcpinfo"
)

func handleConn(conn net.Conn) {
	io.Copy(ioutil.Discard, conn)
}

func server(wg *sync.WaitGroup) {
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	wg.Done()

	// accept connection on port
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleConn(conn)
	}
}

func client() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		panic(err)
	}

	go io.Copy(ioutil.Discard, conn)

	_, err = conn.Write([]byte("hihihihihihihi"))
	if err != nil {
		panic(err)
	}

	//tcpInfo, err := tcpinfo.GetsockoptTCPInfo(conn.(*net.TCPConn))
	tcpInfo, err := tcpinfo.GetTCPInfo(conn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", tcpInfo)
}

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go server(&wg)
	wg.Wait()
	client()
	//==============
	wg.Add(1)
	go tlsServer(&wg)
	wg.Wait()
	tlsClient()
}

func tlsServer(wg *sync.WaitGroup) {
	cer, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen("tcp", ":8443", config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	wg.Done()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}

func tlsClient() {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "127.0.0.1:8443", conf)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	n, err := conn.Write([]byte("hello\n"))
	if err != nil {
		log.Println(n, err)
		return
	}

	tcpInfo, err := tcpinfo.GetTCPInfo(conn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", tcpInfo)
}
