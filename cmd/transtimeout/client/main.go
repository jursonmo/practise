package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-kratos/examples/helloworld/helloworld"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
	callHTTP()
}

// 两种方式把x-md-global-timeout 设置到reqeust header 里
//1. 使用 mmd.Client() Option, 这种方式是所有的api 请求都默认设置这个超时时间到reqeust header，
//2. metadata.AppendToClientContext; 这种方式可以针对特定的某次请求设置超时，同时会可以覆盖第一种设置超时参数,
//  因为mmd.Client() 先执行Option 设置header, 然后从metadata.FromClientContext(ctx) 读到metadata.AppendToClientContext设置超时到header，
//3. 第三种方法，就是写个中间件，把每次请求用的ctx 的 deadline 读出来，设置到reqeust header 里：ClientTransmitTimeout()
//    这种做法不错，像grpc 那样无感知的实现超时传递

//WithConstants(md metadata.Metadata)
func callHTTP() {
	md := metadata.Metadata{"x-md-global-timeout": "800ms"}
	_ = md

	conn, err := http.NewClient(
		context.Background(),
		http.WithMiddleware(
			mmd.Client( /*mmd.WithConstants(md)*/ ), //第一种方法：
			ClientTransmitTimeout(),                 //第三种方法：
		),
		http.WithEndpoint("127.0.0.1:8000"),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := helloworld.NewGreeterHTTPClient(conn)
	t := time.Millisecond * 400
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	//第二种方法：
	//ctx = metadata.AppendToClientContext(ctx, "x-md-global-timeout", "500ms") // 这个会在mmd.Client() 那里读出来，设置到reqeust header
	reply, err := client.SayHello(ctx, &helloworld.HelloRequest{Name: "kratos"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[http] SayHello %s\n", reply)
}

func ClientTransmitTimeout() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				header := tr.RequestHeader()
				if d, ok := ctx.Deadline(); ok {
					timeout := time.Until(d)
					if timeout > 0 {
						header.Set("x-md-global-timeout", fmt.Sprintf("%v", timeout))
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
