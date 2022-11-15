package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/jursonmo/practise/pkg/udp"
)

var (
	listenNum = flag.Int("l", 1, "reuseport listen num")
	dialNum   = flag.Int("d", 1, "udp clients num")
	writeNum  = flag.Int("w", 2, "one udp client write data times")
)

func main() {
	flag.Parse()
	err := udp.ListenReuseport(context.Background(), "udp", "0.0.0.0:2222", *listenNum)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)

	data := []byte("12345678")
	log.Printf("write data:%s, num:%d\n", string(data), *writeNum)

	udp.DialAndWriteBatch("udp", "127.0.0.1:2222", data, *writeNum)

	time.Sleep(time.Hour)
}

/*

root@ubuntu:~# ./main -w 2
2022/11/16 01:21:01 index:0, listen packet at [::]:2222

2022/11/16 01:21:02 write data:12345678, num:2
2022/11/16 01:21:02 write 2 packet //发送了两个报文，每个报文都是1234567812345678
2022/11/16 01:21:03 listen packet at [::]:2222 start reading....
2022/11/16 01:21:03 index:0, got n:2, len(ms):2 //接受了两个报文
2022/11/16 01:21:03 i:0, from addr:127.0.0.1:43388, ms.N:10 //接受到第一个报文的长度是10，发生了截取
2022/11/16 01:21:03 ms[0].Buffers[0] = 12345 //接受内存buffer[0] 的大小就是5，所以会读到“12345”
2022/11/16 01:21:03 ms[0].Buffers[1] = 67812 //接受内存buffer[1] 的大小也是5, 会读取后面的67812，剩下的345678被丢弃了
2022/11/16 01:21:03 i:1, from addr:127.0.0.1:43388, ms.N:12
2022/11/16 01:21:03 ms[1].Buffers[0] = 123456781234 // ms[1].Buffers[0]的长度是12，所以最多读到12个字节，剩下的5678被丢弃
*/
