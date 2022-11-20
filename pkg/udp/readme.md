####  提高 udp 性能 by golang
1. reuseport 特性，让 udp server 可以在同一个地址创建多个listener, 充分利用多cpu
2. 利用ipv4.PacketConn 对底层recvmmsg/sendmmsg 接口的封装，实现批量读写数据，减少udp 读写时发生的系统调用，提高性能
3. 为了让应用层调用的方便，实现类似于bufio的接口( tcp 就是利用原生bufio 达到减少系统调用的目的)

### todo
1. 减少内存copy 次数