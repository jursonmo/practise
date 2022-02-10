package main

import (
	"context"

	"github.com/networkop/xdp-xconnect/pkg/xdp"
)

func main() {

	input := map[string]string{"eth1": "tap1"}
	ctx, _ := context.WithCancel(context.Background())
	app, _ := xdp.NewXconnectApp(input)
	// handle error

	updateCh := make(chan map[string]string, 1)

	app.Launch(ctx, updateCh)
}
