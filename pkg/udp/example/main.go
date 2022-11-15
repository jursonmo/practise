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
	for i := 0; i < *dialNum; i++ {
		udp.Dial("udp", "127.0.0.1:2222", data, *writeNum)
	}
	time.Sleep(time.Hour)
}

/*
root@ubuntu:~# ./main -w 2
2022/11/16 00:27:46 index:0, listen packet at [::]:2222
2022/11/16 00:27:47 write data:12345678, num:2
2022/11/16 00:27:47 conn local:127.0.0.1:43770, i:0, write data len:8
2022/11/16 00:27:47 conn local:127.0.0.1:43770, i:1, write data len:8
2022/11/16 00:27:48 listen packet at [::]:2222 start reading....
2022/11/16 00:27:48 index:0, got n:2, len(ms):2
2022/11/16 00:27:48 i:0, from addr:127.0.0.1:43770, ms.N:8
2022/11/16 00:27:48 ms[0].Buffers[0] = 12345
2022/11/16 00:27:48 ms[0].Buffers[1] = 678
2022/11/16 00:27:48 i:1, from addr:127.0.0.1:43770, ms.N:8
2022/11/16 00:27:48 ms[1].Buffers[0] = 12345678

//用reuseport 启动两个listener 来侦听同一个地址
root@ubuntu:~# ./main -l 2 -d 2 -w 2
2022/11/16 00:34:41 index:1, listen packet at [::]:2222
2022/11/16 00:34:41 index:0, listen packet at [::]:2222
2022/11/16 00:34:42 write data:12345678, num:2
2022/11/16 00:34:42 conn local:127.0.0.1:38362, i:0, write data len:8
2022/11/16 00:34:42 conn local:127.0.0.1:38362, i:1, write data len:8
2022/11/16 00:34:42 conn local:127.0.0.1:57989, i:0, write data len:8
2022/11/16 00:34:42 conn local:127.0.0.1:57989, i:1, write data len:8
2022/11/16 00:34:43 listen packet at [::]:2222 start reading....
2022/11/16 00:34:43 index:1, got n:2, len(ms):2  //这里是关键，说明是1号listener 读到的数据
2022/11/16 00:34:43 i:0, from addr:127.0.0.1:38362, ms.N:8
2022/11/16 00:34:43 ms[0].Buffers[0] = 12345
2022/11/16 00:34:43 ms[0].Buffers[1] = 678
2022/11/16 00:34:43 i:1, from addr:127.0.0.1:38362, ms.N:8
2022/11/16 00:34:43 listen packet at [::]:2222 start reading....
2022/11/16 00:34:43 index:0, got n:2, len(ms):2 //这里是关键，说明是0号listener 读到的数据
2022/11/16 00:34:43 ms[1].Buffers[0] = 12345678
2022/11/16 00:34:43 i:0, from addr:127.0.0.1:57989, ms.N:8
2022/11/16 00:34:43 ms[0].Buffers[0] = 12345
2022/11/16 00:34:43 ms[0].Buffers[1] = 678
2022/11/16 00:34:43 i:1, from addr:127.0.0.1:57989, ms.N:8
2022/11/16 00:34:43 ms[1].Buffers[0] = 12345678
*/
