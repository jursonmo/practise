package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elazarl/goproxy/ext/auth"

	"github.com/elazarl/goproxy"
)

const (
	ProxyAuthHeader = "Proxy-Authorization"
)

func SetBasicAuth(username, password string, req *http.Request) {
	req.Header.Set(ProxyAuthHeader, fmt.Sprintf("Basic %s", basicAuth(username, password)))
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func GetBasicAuth(req *http.Request) (username, password string, ok bool) {
	auth := req.Header.Get(ProxyAuthHeader)
	if auth == "" {
		return
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func main() {
	username, password := "foo", "bar"

	// start end proxy server
	endProxy := goproxy.NewProxyHttpServer()
	endProxy.Verbose = true
	auth.ProxyBasic(endProxy, "my_realm", func(user, pwd string) bool {
		return user == username && password == pwd
	})
	log.Println("serving end proxy server at localhost:8082")
	go http.ListenAndServe("localhost:8082", endProxy)

	// start middle proxy server
	middleProxy := goproxy.NewProxyHttpServer()
	middleProxy.Verbose = true
	//mo: 如果middleProxy 收到GET 请求，会根据Transport Proxy 把request发送给代理
	middleProxy.Tr.Proxy = func(req *http.Request) (*url.URL, error) {
		return url.Parse("http://localhost:8082")
	}
	connectReqHandler := func(req *http.Request) {
		SetBasicAuth(username, password, req)
	}
	//mo: 如果是middleProxy 收到CONNECT 请求，会调用ConnectDial 向下一个代理器建立tcp 连接，
	//然后给client 回应http 200, 同时给下一个代理器发送CONNECT request,
	// 只不过request HOST 的目的地址依然是target, 这样下一个代理器收到也是CONNECT 请求，然后进行类似的处理。
	// 下一个代理器如果是end proxy, 那么它不需要设置ConnectDial，这样就用end proxy.Tr.Dial或者net.Dial 跟target 建立tcp 连接
	// 这样client 跟 target 就建立通道了，可以进行tls 握手
	// 创建Proxy 时，默认去找环境变量https_proxy 来初始化 ConnectDial，没有环境变量，ConnectDial就为nil
	// 这里是手动设置ConnectDial，指向end proxy.
	middleProxy.ConnectDial = middleProxy.NewConnectDialToProxyWithHandler("http://localhost:8082", connectReqHandler)
	middleProxy.OnRequest().Do(goproxy.FuncReqHandler(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		SetBasicAuth(username, password, req)
		return req, nil
	}))
	log.Println("serving middle proxy server at localhost:8081")
	go http.ListenAndServe("localhost:8081", middleProxy)

	time.Sleep(1 * time.Second)

	// fire a http request: client --> middle proxy --> end proxy --> internet
	proxyUrl := "http://localhost:8081"
	request, err := http.NewRequest("GET", "https://ip.cn", nil)
	if err != nil {
		log.Fatalf("new request failed:%v", err)
	}
	tr := &http.Transport{Proxy: func(req *http.Request) (*url.URL, error) { return url.Parse(proxyUrl) }}
	client := &http.Client{Transport: tr}
	rsp, err := client.Do(request)
	if err != nil {
		log.Fatalf("get rsp failed:%v", err)

	}
	defer rsp.Body.Close()
	data, _ := ioutil.ReadAll(rsp.Body)

	if rsp.StatusCode != http.StatusOK {
		log.Fatalf("status %d, data %s", rsp.StatusCode, data)
	}

	log.Printf("rsp:%s", data)
}
