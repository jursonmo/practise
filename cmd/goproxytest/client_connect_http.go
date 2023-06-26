package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// 这里手动指明CONNECT 方法，可以指定target 为http 服务器
// 如果用原生Transport Proxy 的方式，是做不到target 为http，发起的CONNECT
// 因为Transport Proxy 的方式，如果target 是 http, 那么就向代理器发起GET 请求，
//如果target 是https, 就向代理器发起CONNECT 请求

//github.com/elazarl/goproxy/https.go
//copy from goproxy function NewConnectDialToProxyWithHandler
// goproxy/examples/cascadeproxy/main.go  如何使用ConnectDial
func main() {
	network := "tcp"
	addr := "127.0.0.1:1313"

	proxyURL, _ := url.Parse("https://127.0.0.1:8080")
	proxyURL.Opaque = addr
	connectReq := &http.Request{
		Method: "CONNECT",
		//URL:    &url.URL{Opaque: addr},
		URL:    proxyURL,
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

	//现在需要发送正在的target request, 由于代理器已经跟target server 建立连接
	//这里target request HOST 写什么ip 和端口都可以，主要是 path 即，/echo
	//targetURL, _ := url.Parse("http://127.0.0.1:1313/echo")
	targetURL, _ := url.Parse("http://1.1.1.1:1234/echo")
	targetReq := &http.Request{
		Method: "Get",
		URL:    targetURL,
		Host:   targetURL.Host,
		Header: make(http.Header),
	}

	targetReq.Write(c)
	targetResp, err := http.ReadResponse(br, targetReq)
	if err != nil {
		c.Close()
		return
	}
	defer targetResp.Body.Close()
	targetRespData, err := ioutil.ReadAll(targetResp.Body)
	fmt.Printf("%s\n", targetRespData)
	return
}

//practise fail
func main1() {
	// proxyURL, _ := url.Parse("http://proxy.example.com:8080")
	// targetURL, _ := url.Parse("http://example.com")
	proxyURL, _ := url.Parse("http://127.0.0.1:8080")
	targetURL, _ := url.Parse("http://127.0.0.1:1313/echo")
	// 建立到代理服务器的连接
	proxyConn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		fmt.Println("无法连接到代理服务器:", err)
		return
	}
	fmt.Printf("targetURL:%#v\n", targetURL)
	fmt.Printf("proxyURL:%#v\n", proxyURL)
	// 发起CONNECT请求
	request := &http.Request{
		Method: http.MethodConnect,
		URL:    targetURL,
		Host:   targetURL.Host,
		Header: make(http.Header),
	}
	fmt.Printf("request:%#v\n", request)
	req, _ := http.NewRequest(http.MethodConnect, "http://127.0.0.1:1313/echo", nil)
	fmt.Printf("req:%#v\n", req)
	req2 := &http.Request{
		Method: http.MethodGet,
		URL:    proxyURL,
		Host:   proxyURL.Host,
		Header: make(http.Header),
	}
	fmt.Printf("req2:%#v\n", req2)
	// ---------------------------
	tr := &http.Transport{
		//Proxy: http.ProxyFromEnvironment,
		Proxy: http.ProxyURL(proxyURL),
		//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		//Transport: tr,
		Timeout: time.Second * 5, //超时时间
	}
	_ = tr
	resp, err := client.Do(req2)
	time.Sleep(time.Second * 2)
	_ = resp
	return
	// ----------------------------------

	err = request.WriteProxy(proxyConn)
	if err != nil {
		fmt.Println("无法发送CONNECT请求:", err)
		return
	}

	// 读取代理服务器的响应
	response, err := http.ReadResponse(bufio.NewReader(proxyConn), request)
	if err != nil {
		fmt.Println("无法读取代理服务器的响应:", err)
		return
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("代理连接失败:", response.Status)
		return
	}

	fmt.Println("代理连接已建立")

	// 在此处进行后续操作，可以使用proxyConn进行数据传输

	proxyConn.Close()
}
