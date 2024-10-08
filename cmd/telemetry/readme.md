起码要含有时间戳、起止时间、Trace 的 ID、当前 Span 的 ID、父 Span 的 ID 等能够满足追踪需要的信息。

一个 REST 调用或者数据库操作等，都可以作为一个 span 。 span 是分布式追踪的最小跟踪单位，一个 Trace 由多段 Span 组成。

```
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```
通过 http://localhost:16686 可以在浏览器查看 Jaeger UI