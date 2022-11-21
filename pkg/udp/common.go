package udp

var defaultBatchs = 8

//个人觉得, udp 应用层不应该发送超过mtu 1500的报文, 那样导致ip分片，
//丢任意一个分片都导致整个报文丢弃，特别是udp, 运营商特别容易丢弃udp的报文。
var defaultMaxPacketSize = 1600
