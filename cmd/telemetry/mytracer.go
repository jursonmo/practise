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
			semconv.ServiceNameKey.String("mytracer-service"), //es process.serviceName="mytracer-service"
			attribute.String("environment", "development"),    //jaeger Process显示
			attribute.Int64("ID", 1),                          //jaeger Process显示
		)),
	)
	return tp, nil
}

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
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(carrier))
	// Extract 后，可以通过trace.SpanFromContext(ctx) 来获取Span
	//但是carrier.Get(traceparentHeader) 为空，返回原来的ctx。 如果有则生成一个remote==true的SpanContext
	noonSpan := trace.SpanFromContext(ctx) //noonSpan.SpanContext()返回空的trace.SpanContext 结构
	rsc := noonSpan.SpanContext()
	log.Printf("noonSpan:%+v", rsc)

	mytp := otel.GetTracerProvider()
	tracer := mytp.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //jaeger上效果tags: otel.library.name: myTracer-node1,
	//tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //可代替上面两条语句。

	//go.opentelemetry.io/otel/sdk@v1.2.0/trace/tracer.go
	ctx, span := tracer.Start(ctx, nodeName) //传入span name; 生成span 和携带span的ctx; es operationName=node1

	span.AddEvent("testAddEvent", trace.WithAttributes(attribute.String("eventKey", "eventValue"))) //jaeger显示为Logs: evnet=testAddEvent, eventKey=eventValue
	defer span.End()

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
	spantp := span.TracerProvider()
	if mytp == spantp {
		log.Printf("mytp == spantp \n")
	} else {
		log.Printf("mytp != spantp \n")
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

	//模拟异步调用两个下游服务
	//node2:
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		node("node2", carrier)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		node("node3", carrier)
		wg.Done()
	}()
	wg.Wait()
	//模拟network delay
	time.Sleep(time.Millisecond * 100)

}

func node(nodeName string, c map[string][]string) {
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

	fromSpan := trace.SpanFromContext(ctx)
	fsc := fromSpan.SpanContext()
	if !fsc.IsValid() {
		log.Printf("parent Span from remote is in Valid:%+v", fsc)
	}
	log.Printf("parent Span from remote:%+v", fsc)

	//tp := otel.GetTracerProvider()
	//tracer := tp.Tracer(fmt.Sprintf("myTracer-%s", nodeName))
	//如果是相同的名称，那么得到同一个Tracer,会有啥影响
	tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName))  //jaeger上效果tags: otel.library.name: myTracer-node2,
	ctx, span := tracer.Start(ctx, nodeName)                     //传入span name;在parent traceID的情况下生成新SpanID
	span.SetAttributes(attribute.Key("testset").String("value")) //jaeger上效果tags: testset=value
	time.Sleep(time.Millisecond * 50)
	span.SetName(nodeName + "spanName") //这里可以重新设置span name
	span.End()

	//打印span 信息，包含 traceID spanID
	sc := span.SpanContext()
	log.Printf("%s current spanContext:%+v\n", nodeName, sc)

	//打印 traceID spanID，traceID 都一样，span id 是不一样的
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(carrier),
	)
	fmt.Printf("node:%s, inject carrier:%v\n", nodeName, carrier)

	//模拟调用集群内其他子服务的延迟
	time.Sleep(time.Millisecond * 50)
	ctx, childSpan := tracer.Start(ctx, nodeName+"sub1")
	//模拟子服务处理花费的时间
	time.Sleep(time.Millisecond * 50)
	childSpan.End()
}

/*
MBP:telemetry obc$ ./mytracer
2022/03/08 18:47:54 noonSpan:{traceID:[0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0] spanID:[0 0 0 0 0 0 0 0] traceFlags:0 traceState:{list:[]} remote:false}
2022/03/08 18:47:54 node1, rootSpan:{traceID:[154 38 168 238 136 39 112 138 198 153 147 204 151 29 82 168] spanID:[115 159 68 94 191 83 112 151] traceFlags:1 traceState:{list:[]} remote:false}
2022/03/08 18:47:54 span == rootSpan
2022/03/08 18:47:54 mytp == spantp
node:node1, inject carrier:map[Traceparent:[00-9a26a8ee8827708ac69993cc971d52a8-739f445ebf537097-01]]
2022/03/08 18:47:54 calling node3 services
2022/03/08 18:47:54 calling node2 services
2022/03/08 18:47:54 parent Span from remote:{traceID:[154 38 168 238 136 39 112 138 198 153 147 204 151 29 82 168] spanID:[115 159 68 94 191 83 112 151] traceFlags:1 traceState:{list:[]} remote:true}
2022/03/08 18:47:54 parent Span from remote:{traceID:[154 38 168 238 136 39 112 138 198 153 147 204 151 29 82 168] spanID:[115 159 68 94 191 83 112 151] traceFlags:1 traceState:{list:[]} remote:true}
2022/03/08 18:47:54 node3 current spanContext:{traceID:[154 38 168 238 136 39 112 138 198 153 147 204 151 29 82 168] spanID:[150 151 4 61 140 30 113 64] traceFlags:1 traceState:{list:[]} remote:false}
node:node3, inject carrier:map[Traceparent:[00-9a26a8ee8827708ac69993cc971d52a8-9697043d8c1e7140-01]]
2022/03/08 18:47:54 node2 current spanContext:{traceID:[154 38 168 238 136 39 112 138 198 153 147 204 151 29 82 168] spanID:[116 128 40 160 108 94 73 215] traceFlags:1 traceState:{list:[]} remote:false}
node:node2, inject carrier:map[Traceparent:[00-9a26a8ee8827708ac69993cc971d52a8-748028a06c5e49d7-01]]
2022/03/08 18:47:54 Post "http://localhost:14268/api/traces": dial tcp [::1]:14268: connect: connection refused
*/

/* --------es----------

_id
r7HT8H0BWfcfyQID6_Tg

_index
jaeger-span-2021-12-25

_score
 -

_type
_doc

duration
359,455

flags
1

logs

{
  "fields": [
    {
      "type": [
        "string"
      ],
      "value": [
        "testAddEvent"
      ],
      "key": [
        "event"
      ]
    },
    {
      "type": [
        "string"
      ],
      "value": [
        "eventValue"
      ],
      "key": [
        "eventKey"
      ]
    }
  ],
  "timestamp": [
    1640422953358130
  ]
}

operationName
node1

process.serviceName
mytracer-service

process.tag.environment
development

process.tag.ID
1

spanID
f5af296c76e76895

startTime
1,640,422,953,358,081

startTimeMillis
Dec 25, 2021 @ 17:02:33.358

tag.internal@span@format
jaeger

tag.otel@library@name
myTracer-node1

traceID
0606134956bbb054bdfd5d0fe395eb84
*/
