### 业务需求，实时读取最新状态信息，同时可以查看状态信息的历史记录。通过接口可以读到状态消息，但是无法实时获取，不能一直循环读取接口，这样消耗网络资源，同时也不够实时。需要增加一个通知机制，即状态变化时，发送通知消息。最容易想到的是用redis sub/pub 订阅和发布机制来起到通知状态消息的效果，但是redis sub/pub 无法持久化，如果查看历史状态消息，如果业务程序挂了，就不能获取这段时间内“状态消息”的变化过程，可以用redis sub/pub + redis list , 让list 来保证消息历史记录的持久化。 但是这样写起代码比较麻烦一些，redis 5.0 以后的版本增加了 stream 的功能， 可以同时满足消息通知和消息持久化的需求。状态有变化时，把状态消息写入stream, 消费就能实时读取到这个“状态消息”

#### 步骤：
1. 先用 xrevrange 读取 stream 最新的消息
 1.1  如果stream里没有消息，直接去读接口拿到最新状态信息。
 1.2  如果stream读到最新消息，可以按这个消息作为最新状态信息来处理
2. 阻塞式等待stream 的最新消息
   2.1 如果之前读stream，没有读到消息(stream 没有任何消息)， 根据"$" 来塞读取stream？
       （如果从第一步读stream, 到这里之前的间隙，已经产生了新的消息，那么用“$” 来读，就读不到这个消息了，是不是应该用“0” 来, 是的)
   2.2 如果之前有读到消息， 根据消息的ID 来阻塞等待stream的新消息, 不会错过任何消息。