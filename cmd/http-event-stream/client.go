package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	// 创建HTTP客户端
	client := &http.Client{}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", "http://localhost:8080/events", nil)
	if err != nil {
		panic(err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// 发送HTTP请求
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 读取并处理服务器发送的事件
	decoder := NewEventDecoder(resp.Body)
	for {
		event, err := decoder.Decode()
		if err != nil {
			panic(err)
		}
		fmt.Printf("data:%v\n", event.Data)
	}
}

// 定义事件结构体
type Event struct {
	ID    string
	Event string
	Data  string
}

// 定义事件解码器
type EventDecoder struct {
	reader *bufio.Reader
}

func NewEventDecoder(reader io.Reader) *EventDecoder {
	return &EventDecoder{bufio.NewReader(reader)}
}

func (decoder *EventDecoder) Decode() (*Event, error) {
	event := &Event{}
	var err error

	for {
		// 读取一行
		var line []byte
		//服务器发的数据data: 开头， \n\n 结尾， 有两个\n,
		line, err = decoder.reader.ReadBytes('\n')
		//line, err = decoder.reader.ReadString("\n\n")
		if err != nil {
			return nil, err
		}

		// 去掉行末的换行符
		line = bytes.TrimSpace(line)
		//fmt.Println(line)
		// 解析行
		if len(line) == 0 {
			// 服务器发的数据data: 开头， \n\n 结尾， 有两个\n, 所以这里会解释出空行。
			fmt.Println("忽略空行")
			continue
		} else if line[0] == ':' {
			// 忽略注释行
			continue
		}
		content := string(line)
		if strings.HasPrefix(content, "id:") {
			event.ID = strings.TrimSpace(string(line[3:]))
		} else if strings.HasPrefix(content, "event:") {
			event.Event = strings.TrimSpace(string(line[6:]))
		} else if strings.HasPrefix(content, "data:") {
			//每个事件由类型(event)和数据(data)两部分组成，同时每个事件可以有一个可选的标识符(id)。
			//读到data: 就应该返回了
			event.Data = strings.TrimSpace(string(line[5:]))
			return event, nil
		}
	}
}
