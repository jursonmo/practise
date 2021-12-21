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
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("mytracer-service"),
			attribute.String("environment", "development"),
			attribute.Int64("ID", 1),
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
	//但是carrier.Get(traceparentHeader) 为空，返回原来的ctx
	noonSpan := trace.SpanFromContext(ctx) //noonSpan.SpanContext()返回空的trace.SpanContext 结构
	rsc := noonSpan.SpanContext()
	log.Printf("noonSpan:%+v", rsc)

	mytp := otel.GetTracerProvider()
	tracer := mytp.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //jaeper上效果 otel.library.name: myTracer-node1,
	//tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName))
	ctx, span := tracer.Start(ctx, nodeName) //传入span name; 生成span 和携带span的ctx
	defer span.End()
	rootSpan := trace.SpanFromContext(ctx)
	rsc = rootSpan.SpanContext()
	log.Printf("rootSpan:%+v", rsc)

	//检查span ,跟ctx的span 是否一样
	if span == rootSpan {
		log.Printf("span == rootSpan \n")
	} else {
		log.Fatal("span != rootSpan ")
	}

	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(carrier),
	)
	//go.opentelemetry.io/otel@v1.2.0/propagation/trace_context.go
	//carrier 格式为 %.2x-%s-%s-%s:supportedVersion-TraceID-SpanID-flags
	fmt.Printf("node:%s, carrier:%v\n", nodeName, carrier)

	//模拟network delay
	time.Sleep(time.Millisecond * 100)

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
	log.Printf("calling %s\n", nodeName)
	carrier := c //get carrier from node1
	ctx := context.Background()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(carrier))

	fromSpan := trace.SpanFromContext(ctx)
	fsc := fromSpan.SpanContext()
	log.Printf("fromSpan:%+v", fsc)

	//tp := otel.GetTracerProvider()
	//tracer := tp.Tracer(fmt.Sprintf("myTracer-%s", nodeName))
	//如果是相同的名称，那么得到同一个Tracer,会有啥影响
	tracer := otel.Tracer(fmt.Sprintf("myTracer-%s", nodeName)) //jaeper上效果 otel.library.name: myTracer-node2,
	ctx, span := tracer.Start(ctx, nodeName)                    //传入span name
	span.SetAttributes(attribute.Key("testset").String("value"))
	time.Sleep(time.Millisecond * 50)
	span.SetName(nodeName + "spanName") //这里可以重新设置span name
	span.End()

	//打印span 信息，包含 traceID spanID
	sc := span.SpanContext()
	log.Printf("%+v", sc)

	//打印 traceID spanID，traceID 都一样，span id 是不一样的
	otel.GetTextMapPropagator().Inject(ctx,
		propagation.HeaderCarrier(carrier),
	)
	fmt.Printf("node:%s, carrier:%v\n", nodeName, carrier)

	time.Sleep(time.Millisecond * 50)
	ctx, childSpan := tracer.Start(ctx, nodeName+"-1")
	time.Sleep(time.Millisecond * 50)
	childSpan.End()
}
