package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-kratos/examples/helloworld/helloworld"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "helloworld"
	// Version is the version of the compiled software.
	// Version = "v1.0.0"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	helloworld.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	var clientTimeout time.Duration
	var cancel context.CancelFunc
	/*
		var err error
		var mdTimeout string
		if md, ok := metadata.FromServerContext(ctx); ok {
			mdTimeout = md.Get("x-md-global-timeout")
			fmt.Printf("get metadata x-md-global-timeout:%s\n", mdTimeout)
			if mdTimeout != "" {
				clientTimeout, err = time.ParseDuration(mdTimeout)
				if err != nil {
					fmt.Println(err)
					return nil, errors.New(500, "ParseDuration err", fmt.Sprintf("x-md-global-timeout %s invalid", mdTimeout))
				}
			}
		}


		if d, ok := ctx.Deadline(); ok {
			fmt.Printf("当前服务默认超时时间是:%v, client 传过来的超时时间是:%v\n", time.Until(d), clientTimeout)
			if clientTimeout > 0 && clientTimeout < time.Until(d) {
				ctx, cancel = context.WithTimeout(ctx, clientTimeout)
				defer cancel()
			}
		}
	*/

	clientTimeout, ctx, cancel = shrinkTimeoutCtx(ctx)
	defer cancel()

	if clientTimeout != 0 {
		//这里处理业务逻辑，并可以调用下一个服务的接口，但是调用下一个服务前，需要把当前的超时时间设置到 metadata  x-md-global-timeout 里
		//1. do something
		start := time.Now()
		doSomethingLocal(ctx)
		cost := time.Since(start)
		fmt.Printf("本地处理逻辑花费时间为%v\n", cost)
		//2. 把剩余的时间 设置到 metadata  x-md-global-timeout 里
		left := clientTimeout - cost
		if left <= 0 {
			return nil, errors.New(500, "xxx", "no time for calling next service")
		}
		fmt.Printf("留给调用下一个服务的剩余时间:%v\n", left)
		var cancelnext context.CancelFunc
		ctx, cancelnext = context.WithTimeout(ctx, left)
		defer cancelnext()
		ctx = metadata.AppendToClientContext(ctx, "x-md-global-timeout", fmt.Sprintf("%v", left))
		callNextService(ctx)

		//调用下一个服务返回后，还剩余多少时间
		left = clientTimeout - time.Since(start)
		fmt.Printf("回应 client 时，剩余时间:%v\n", left)
		return &helloworld.HelloReply{Message: fmt.Sprintf("Hello %s timeout left: %s", in.Name, left)}, nil
	} else {
		fmt.Printf("没有设置超时时间, 直接处理\n")
		doSomethingLocal(ctx)
		callNextService(ctx)
		return &helloworld.HelloReply{Message: fmt.Sprintf("Hello %s  unset timeout", in.Name)}, nil
	}
}

//返回还有多少时间timeout, 和 ctx
func shrinkTimeoutCtx(ctx context.Context) (time.Duration, context.Context, context.CancelFunc) {
	var mdTimeout string
	var clientTimeout time.Duration

	if md, ok := metadata.FromServerContext(ctx); ok {
		mdTimeout = md.Get("x-md-global-timeout")
		fmt.Printf("get metadata x-md-global-timeout:%s\n", mdTimeout)
		if mdTimeout != "" {
			clientTimeout, _ = time.ParseDuration(mdTimeout)
		}
	}

	var cancel context.CancelFunc
	if d, ok := ctx.Deadline(); ok {
		fmt.Printf("当前服务默认超时时间是:%v, client 传过来的超时时间是:%v\n", time.Until(d), clientTimeout)
		if clientTimeout > 0 && clientTimeout < time.Until(d) {
			ctx, cancel = context.WithTimeout(ctx, clientTimeout)
			return clientTimeout, ctx, cancel
		}
		return time.Until(d), ctx, func() {}
	}
	return 0, ctx, func() {}
}

func doSomethingLocal(ctx context.Context) {
	//模拟本地业务逻辑处理：
	time.Sleep(time.Millisecond * 100)
}

func callNextService(ctx context.Context) {
	//模拟调用下一个服务花费的时间：
	time.Sleep(time.Millisecond * 100)
}

func main() {
	// grpcSrv := grpc.NewServer(
	// 	grpc.Address(":9000"),
	// 	grpc.Middleware(
	// 		mmd.Server(),
	// 	))
	httpSrv := http.NewServer(
		http.Address(":8000"),
		http.Middleware(
			mmd.Server(), //把request header x-md- 设置到ctx metadata 里
		),
	)

	s := &server{}
	//helloworld.RegisterGreeterServer(grpcSrv, s)
	helloworld.RegisterGreeterHTTPServer(httpSrv, s)

	app := kratos.New(
		kratos.Name(Name),
		kratos.Server(
			httpSrv,
			//grpcSrv,
		),
	)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
