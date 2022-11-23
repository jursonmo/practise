1. 实现udp net.listener 相关接口，Listen, Accept, 方便应用层使用
2. udp listenr 一个读取报文，把报文交给自定义的对象UDPConn，UDPConn 实现net.Conn的接口方便应用层使用
3. UDPConn 每次系统调用只读写一个数据包，效率低，为了提高效率，利用ipv4.PacketConn 批量读取数据，同时提供类似bufio 的接口，方便应用层使用，例子在useBufioExample 目录下
4. listener 接受读取报文时，也是利用ipv4.PacketConn 批量读取数据，这样即使UDPConn调用read(),也不是每个系统调用值读取一个数据包，实际是listener 批量读，批量放到UDConn rxqueue里。
5. listener 批量读时，再放到UDPConn rxqueue里，这里需要创建新的内存对象，同时copy 操作一次，这样不利于对象复用。所以实现bufferpool, 实现 readLoopv2 实现 批量读和对象复用。
6. 前面说了，服务器accpet 生成的UDPConn，在UDPConn 批量写时，其实用的是listener 底层的socket, 也就是多个UDPConn 并发批量写时，其实是由内核锁来互斥的，这个是没有问题的，但是感觉不是很正规，一般的做法是，有一个线程负责发送一个socket 的数据，也就是应该把多个UDPConn的数据放在一个队列里，由一个任务取队列的数据，然后批量发送。这个在lnwritebatch.go 里实现。这样服务器accpet 生成UDPConn可以不用bufio,简单调用Write(),底层也是批量写的.
7. 2022-11-23为止，服务器accpet 生成UDPConn 简单调用Read()\Write()，底层都是listener socket 批量读写的。 client dial 生成的UDPConn, 如果要批量读写，还是要用bufio