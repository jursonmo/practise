####  提高 udp 性能 by golang (only on linux)
1. reuseport 特性，让 udp server 可以在同一个地址创建多个listener, 充分利用多cpu, 同时能减少读写socket 时锁的竞争。
2. 利用ipv4.PacketConn 对底层recvmmsg/sendmmsg 接口的封装，实现批量读写数据，减少udp 读写时发生的系统调用，提高性能。（一次系统调用，读或写多个udp数据包)
3. 为了让应用层调用的方便，实现类似于bufio的接口( tcp 就是利用原生bufio 达到减少系统调用的目的)

### todo
1. 减少内存copy 次数(fixed)
2. 增加控制报文和数据报文类型 (fixed: 控制报文和数据报文分离，可以重新开发通信协议库，比如proto，底层通信用当前库就行，不需要在当前库上开发)
3. 心跳检测(心跳作为控制报文，每次收到心跳可以重置conn.SetReadDeadline(), 提供一个类似于websocket 的SetPingHeadler,SetPongHandler 的接口，这样上层可以wrapHandler 重置ReadDeadline，这样就不需要每个数据都去重置ReadDeadline，因为频繁重置ReadDeadline也是有性能损耗的)


### 注意：
1. 一次系统调用得到多个buffer：
buffer1, buffer2.......buffer8
2. 应用层一次conn Read(p []byte) 从一个buffer 中copy 数据到p, p 必须大于buffer, 否则只能读到部分数据，即必须要一次性读完buffer的数据。

3. 如果应用层用 r = bufio.Reader, r.Read() 期望一次读取bufio底层buf大小的数据 --> conn.Read()-->从一个buffer 中copy 数据， 即bufio.Reader的底层buf 大小不能小于一个udp 报文的大小，也就是不能小于系统调用用的buffer的大小。

当应用层一次conn Read(p []byte) 从一个buffer 中读（copy）数据时，如果没有读完， 最好返回一个short_read 的错误，让应用层能发现问题并且做出相应的决策，否则应用层没能读取到一个完整的udp报文 却完全没意识到。

其实可以支持应用层可以多次读取“系统调用用的buffer”，实现的方法就是，没读完的buffer 先暂存起来，下次继续读，读完的buffer 就释放掉，再从队列里拿新的buffer。
