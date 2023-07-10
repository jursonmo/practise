package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

//用Transport Proxy 的方式，
//如果target 是 http, 那么就向代理器发起GET 请求，
//如果target 是https, 就向代理器发起CONNECT 请求
//其实也很好理解，因为target 是https 的话，需要跟target 进行tls 握手，所以代理器只是起到中转tcp连接的数据即可
//如果target 是 http, 不需要跟target进行tls 协商，也就是代理器可以直接代理请求，即代理器向target 发起http request

// 1. 通过代理 访问 自己http服务器http://127.0.0.1:1313
func main() {
	proxyUrl, _ := url.Parse("http://localhost:8080") // 这里写http, 不能写https, 因为proxy 是侦听http 的

	//target server 是http, 代理器收到request 的方法是GET, 代理器得到target request，再发起请求即可。
	//如果target server 是https, 代理器收到request 的方法是CONNECT, 即使这里设置的方法是GET
	request, err := http.NewRequest("GET", "http://127.0.0.1:1313/echo", nil)
	if err != nil {
		log.Fatalf("new request failed:%v", err)
	}
	tr := &http.Transport{
		//Proxy: func(req *http.Request) (*url.URL, error) { return url.Parse(proxyUrl) },
		Proxy: http.ProxyURL(proxyUrl),
	}

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
