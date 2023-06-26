package main

import (
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	//proxy.ConnectDial 只有出了CONNECT 方法的代理才用得到，
	//middleProxy.ConnectDial = middleProxy.NewConnectDialToProxyWithHandler("http://localhost:8082", connectReqHandler)

	// proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
	// 	HandleConnect(goproxy.AlwaysMitm)
	// proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*:80$"))).
	// 	HijackConnect()
	//proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("reddit.*:443$"))).HandleConnect(goproxy.AlwaysReject)
	// proxy.OnRequest(goproxy.DstHostIs("www.reddit.com")).HandleConnect(YourHandlerFunc())
	//proxy.OnRequest(goproxy.DstHostIs("www.reddit.com")).Do(YourHandlerFunc())

	log.Println("proxy listen on :8080")
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
