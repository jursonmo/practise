package main

import (
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	// proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
	// 	HandleConnect(goproxy.AlwaysMitm)
	// proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*:80$"))).
	// 	HijackConnect()
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
