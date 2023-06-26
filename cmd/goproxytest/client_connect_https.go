package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
)

// 手动指明CONNECT 方法，target 是自定义的https 服务器

//github.com/elazarl/goproxy/https.go
//copy from goproxy function NewConnectDialToProxyWithHandler
// goproxy/examples/cascadeproxy/main.go  如何使用ConnectDial
func main() {
	network := "tcp"
	addr := "127.0.0.1:1443"

	// proxyURL, _ := url.Parse("https://127.0.0.1:8080")
	// proxyURL.Opaque = addr
	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		//URL:    proxyURL,
		Host:   addr,
		Header: make(http.Header),
	}

	c, err := net.Dial(network, "127.0.0.1:8080")
	if err != nil {
		return
	}
	connectReq.Write(c)
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		log.Println(err)
		c.Close()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respData, err := ioutil.ReadAll(resp.Body) //如果服务器没有关闭，这会一直阻塞吗
		if err != nil {
			log.Println(err)
			return
		}
		c.Close()
		fmt.Println("proxy refused connection" + string(respData))
		return
	}
	fmt.Println("proxy ok")
	//到这里表示CONNECT 代理器成功连上target server

	c = tls.Client(c, &tls.Config{InsecureSkipVerify: true}) // target 服务器是自定义的服务器，client 不要检查服务器证书。
	//现在需要发送正在的target request, 由于代理器已经跟target server 建立连接
	//这里target request HOST 写什么ip 和端口都可以，主要是 path 即，/echo
	//targetURL, _ := url.Parse("http://127.0.0.1:1313/echo")
	targetURL, _ := url.Parse("http://127.0.0.1:1443/echo")
	targetReq := &http.Request{
		Method: "Get",
		URL:    targetURL,
		Host:   targetURL.Host,
		Header: make(http.Header),
	}

	targetReq.Write(c)
	br = bufio.NewReader(c) // 这里的c 已经是tls client conn 了， 不是tcp client conn
	targetResp, err := http.ReadResponse(br, targetReq)
	if err != nil {
		fmt.Println(err)
		c.Close()
		return
	}
	defer targetResp.Body.Close()
	fmt.Printf("targetResp statusCode:%v, ContentLength:%d\n", targetResp.StatusCode, targetResp.ContentLength)
	targetRespData, err := ioutil.ReadAll(targetResp.Body)
	fmt.Printf("targetRespData:%s\n", targetRespData)
	return
}
