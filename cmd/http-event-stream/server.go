package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("/events")
		// 设置响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 模拟事件源，每秒钟发送一个事件
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				fmt.Println("Client closed connection")
				return
			case t := <-ticker.C:
				fmt.Printf("time to send\n")
				//SSE 消息需要满足一定的格式要求，比如每个消息必须以 "data:" 开头，以 "\n\n" 结尾
				//，同时可以携带一些可选的事件名和 ID 信息。这些格式要求可以参考相关的文档和规范
				fmt.Fprintf(w, "data:%s\n\n", t.Format(time.RFC3339))
				flusher, ok := w.(http.Flusher)
				if ok {
					flusher.Flush()
				} else {
					fmt.Println("Streaming unsupported!")
				}
			}
		}
	})

	// 启动HTTP服务器
	fmt.Println("Starting server at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
