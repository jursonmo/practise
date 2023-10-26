通信协议格式，可扩展

目标：
1。 底层通信协议抽象：即不管底层是tcp,udp, http, quic 等
2. 可扩展性强， ver, tlv option
3. 注册消息类型（注册路由），即为某类型数据注册回调函数。
4. 发送消息级别：
  + 4.1 发送即可（尽力而为）
  + 4.2 至少到达一次（需要对方确认acK，接受方自己实现幂等性)
5. 同步发送（等待ack 才返回，id, seq），异步发送（传入channel,等待结果通知，有超时）
6. 控制消息和用户数据分离， 默认自动实现心跳，
  + 6.1 只需要让可以传入心跳的配置。内部实现tcp 的心跳机制，比如多少秒发送一次，超时没有收到后，多少秒再发送一次，一共几次算超时，interval, x, probe
  + 6.2 可让用户注册心跳器, 心跳id-心跳器：协议成提供发送心跳的接口，以及心跳的内容是什么
    用户实现的接口 func(ctx, hbSend func(ctx, content []byte)error )
    func SetTimeout(c chan )
    func OnReciveHb(hbPkg chan)//读到心跳回应，就重置计数器，在期限内没收到回应，调用SetTimeout来通知超时，让协议关闭连接
    