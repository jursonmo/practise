package main

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// 1. 通过代理 访问 外部https 服务器 https://ip.cn,
//     效果跟系统设置了https 代理为127.0.0.1:8080，然后在浏览器上访问https://ip.cn 一样
// 2. 通过代理 访问 自己https服务器https://127.0.0.1:1443
func main() {
	proxyUrl := "http://localhost:8080" // 这里写http, 不能写https, 因为proxy 是侦听http 的
	/*
		request, err := http.NewRequest("GET", "https://ip.cn", nil) // proxy 接受到的是CONNECT 的方法
		if err != nil {
			log.Fatalf("new request failed:%v", err)
		}
		// 说明设置了Proxy后，go net 内部会等待http 200 的回应后，再进行tls 协商
		tr := &http.Transport{Proxy: func(req *http.Request) (*url.URL, error) { return url.Parse(proxyUrl) }}
	*/

	//虽然这里写了GET, 但是proxy 接受到的是CONNECT 的方法, 内部会等待代理器返回http 200后，再进行tls协商
	request, err := http.NewRequest("GET", "https://127.0.0.1:1443/echo", nil)
	if err != nil {
		log.Fatalf("new request failed:%v", err)
	}
	tr := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) { return url.Parse(proxyUrl) },
		//由于https://127.0.0.1:1443/echo是自己服务器，证书不是权威机构签名的，所以这里不检查证书的合法性
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
