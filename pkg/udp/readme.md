####  提高 udp 性能 by golang
1. reuseport 特性，让 udp server 可以在同一个地址创建多个listener, 充分利用多cpu, 同时能减少读写socket 时锁的竞争。
2. 利用ipv4.PacketConn 对底层recvmmsg/sendmmsg 接口的封装，实现批量读写数据，减少udp 读写时发生的系统调用，提高性能。（一次系统调用，读或写多个udp数据包)
3. 为了让应用层调用的方便，实现类似于bufio的接口( tcp 就是利用原生bufio 达到减少系统调用的目的)

### todo
1. 减少内存copy 次数(fixed)
2. 增加控制报文和数据报文类型，
3. 心跳检测(心跳作为控制报文，每次收到心跳可以重置conn.SetReadDeadline(), 提供一个类似于websocket 的SetPingHeadler,SetPongHandler 的接口，这样上层可以wrapHandler 重置ReadDeadline，这样就不需要每个数据都去重置ReadDeadline，因为频繁重置ReadDeadline也是有性能损耗的)