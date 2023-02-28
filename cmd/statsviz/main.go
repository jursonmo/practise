package main

import (
	"net/http"

	"github.com/arl/statsviz"
)

func main() {
	mux := http.NewServeMux()
	statsviz.Register(mux)
	http.ListenAndServe("0.0.0.0:6060", mux)
}
