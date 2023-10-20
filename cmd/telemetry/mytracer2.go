package main

//参考：
//https://github.com/open-telemetry/opentelemetry-go/blob/main/example/jaeger/main.go
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Get trace provider
func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		//tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("mytracer-netwrokDelay"),
			attribute.String("environment", "development"),
			attribute.Int64("ID", 1),
		)),
	)
	return tp, nil
}

// 模拟： 用链路跟踪来展示网络延迟时间，而不是程序处理时间
func main() {
	tp, err := tracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		log.Fatal(err)
	}
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil { //必须做
			log.Fatal(err)
		}
	}(ctx)

	//node 1:
	nodeName := "node1"

	carrier := make(map[string][]string)
	//carrier := make(propagation.HeaderCarrier)
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(carrier))
	// Extract 后，可以通过trace.SpanFromContext(ctx) 来获取Span
	//但是carrier.Get(traceparentHeader) 为空，返回原来的ctx。 如果有则生成一个remote==true的SpanContext
	noonSpan := trace.SpanFromContext(ctx) //noonSpan.SpanContext()返回空的trace.SpanContext 结构
	rsc := noonSpan.SpanContext()
	log.Printf("noonSpan:%+v", rsc)

	mytp := otel.GetTracerProvider()
	tracer := mytp.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //jaeger上效果 otel.library.name: myTracer-node1,
	//tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //可代替上面两条语句。

	//go.opentelemetry.io/otel/sdk@v1.2.0/trace/tracer.go
	ctx, span := tracer.Start(ctx, nodeName) //传入span name; 生成span 和携带span的ctx
	//ctx, span := tracer.Start(ctx, nodeName, trace.WithTimestamp(time.Now()))
	//defer span.End()

	//rsc = trace.SpanContextFromContext(ctx) //直接从ctx 提取SpanContext
	rootSpan := trace.SpanFromContext(ctx)
	rsc = rootSpan.SpanContext()
	log.Printf("%s, rootSpan:%+v", nodeName, rsc)

	//检查span ,跟ctx的span 是否一样
	if span == rootSpan {
		log.Printf("span == rootSpan \n")
	} else {
		log.Fatal("span != rootSpan ")
	}

	// 将关于本地追踪调用的span context，设置到carrier(可以是http header)上，并传递出去
	// as trace.SpanKindClient, inject to carrier
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(carrier),
	)
	//go.opentelemetry.io/otel@v1.2.0/propagation/trace_context.go
	//carrier 格式为 %.2x-%s-%s-%s:supportedVersion-TraceID-SpanID-flags
	fmt.Printf("node:%s, inject carrier:%v\n", nodeName, carrier)

	//模拟network delay
	time.Sleep(time.Millisecond * 100)

	//模拟异步调用下游服务
	//node2:
	wg := sync.WaitGroup{}
	wg.Add(1)
	var parentSpanEndTime time.Time
	go func() {
		parentSpanEndTime = node("node2", carrier)
		wg.Done()
	}()
	wg.Wait()

	//模拟network delay
	time.Sleep(time.Millisecond * 100)

	span.End(trace.WithTimestamp(parentSpanEndTime))
}

func node(nodeName string, c map[string][]string) time.Time {
	log.Printf("calling %s services\n", nodeName)
	carrier := make(map[string][]string)
	//get carrier from parent node
	for k, v := range c {
		carrier[k] = v
	}

	ctx := context.Background()
	//as trace.SpanKindServer, Extract from carrier and generate new ctx with remote span
	//Extract will return trace.ContextWithRemoteSpanContext(ctx, sc)
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(carrier)) //carrier有trace信息，Extract则生成一个remote==true的SpanContext

	parentSpan := trace.SpanFromContext(ctx)
	psc := parentSpan.SpanContext()
	log.Printf("parent Span from remote:%+v", psc)
	parentSpanEndTime := time.Now()

	//模拟程序处理时间
	time.Sleep(time.Millisecond * 20)
	//tp := otel.GetTracerProvider()
	//tracer := tp.Tracer(fmt.Sprintf("myTracer-%s", nodeName))
	//如果是相同的名称，那么得到同一个Tracer,会有啥影响

	tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //jaeger上效果 otel.library.name: myTracer-node2,
	ctx, span := tracer.Start(ctx, nodeName)                    //传入span name;在parent traceID的情况下生成新SpanID
	span.SetAttributes(attribute.Key("testset").String("value"))
	span.SetName(nodeName + "spanName") //这里可以重新设置span name

	//打印span 信息，包含 traceID spanID
	sc := span.SpanContext()
	log.Printf("%s current spanContext:%+v\n", nodeName, sc)

	//打印 traceID spanID，traceID 都一样，span id 是不一样的
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(carrier),
	)
	fmt.Printf("node:%s, inject carrier:%v\n", nodeName, carrier)

	//模拟调用集群内其他子服务的网络延迟
	time.Sleep(time.Millisecond * 50)
	span.End() //到了子节点后，span 就完成

	//模拟子服务处理花费的时间
	time.Sleep(time.Millisecond * 20)
	ctx, childSpan := tracer.Start(ctx, nodeName+"sub1")

	//模拟子服务返回到上级的网络延迟
	time.Sleep(time.Millisecond * 50)
	childSpan.End()

	//return parentSpanEndTime, and parent will end()
	return parentSpanEndTime
}
