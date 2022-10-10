package main

import (
	"context"
	"time"

	"github.com/jursonmo/practise/pkg/dial"
)

//GOOS=linux go build main.go
func main() {
	conn, err := dial.Dial(context.Background(), "tcp://127.0.0.1:8080", dial.WithKeepAlive(time.Second*5))
	_ = conn
	_ = err
}
