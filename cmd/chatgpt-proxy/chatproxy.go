package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/jursonmo/practise/pkg/dial"
	"github.com/sashabaranov/go-openai"
)

//todo:
//1. 程序参数指定 key(fixed)
//2. 限制 每次会话的msg(fixed), 打印 token 花费(fixed)
//3. 限制请求的次数，查看相关文档https://platform.openai.com/docs/guides/rate-limits/overview
//4. ws
//5. 通信格式。

/*
在OpenAI的官方聊天平台chat.openai.com上，默认的max_tokens参数设置为50。
这意味着，当您在聊天平台上输入一条消息并按下回车键时，系统将使用最多50个tokens来生成自动回复。

值得注意的是，chat.openai.com并不是通过OpenAI API来提供聊天服务的。
该平台使用了一种名为GPT-3微调模型（GPT-3 fine-tuned model）的特殊模型，
该模型是在OpenAI GPT-3模型的基础上进行微调的。
因此，chat.openai.com的默认设置可能与使用OpenAI API的应用程序不同。

如果您使用OpenAI API来生成自动完成结果，则默认的max_tokens参数取决于您所使用的API模型和计划。
在使用API时，请务必查看相应文档以了解API的默认设置。
*/

/*
{
   "id":"chatcmpl-abc123",
   "object":"chat.completion",
   "created":1677858242,
   "model":"gpt-3.5-turbo-0301",
   "usage":{
      "prompt_tokens":13,
      "completion_tokens":7,
      "total_tokens":20
   },
   "choices":[
      {
         "message":{
            "role":"assistant",
            "content":"\n\nThis is a test!"
         },
         "finish_reason":"stop",
         "index":0
      }
   ]
}

prompt_tokens 输入的 token 数量，
completion_tokens 是 ChatGPT 回复的 token 数量，
total_tokens 是总共使用的 token 数量
*/

var DefaultMaxTokens = 40
var DefaultRecordMsgs = 3 * 2 //保留三次对话的消息
var HeaderSize = 2

type chatObject struct {
	msgs []openai.ChatCompletionMessage
	conn net.Conn
}

type ChatClient struct {
	conn     net.Conn
	aiClient *openai.Client
	chatMsgs []openai.ChatCompletionMessage
}

const (
	CLIENT = "client"
	SERVER = "server"
	PROXY  = "proxy"
)

var (
	mode       = flag.String("m", CLIENT, "client: -r tcp://x.x.x.x:{port}, proxy: -l -r, server: -l ")
	remoteAddr = flag.String("r", "tcp://127.0.0.1:1420", "client/proxy mode, the addr connect to ")
	localAddr  = flag.String("l", "tcp://0.0.0.0:1420", "server/proxy mode, the listen addr ")
	key        = flag.String("k", "", "openai key")
	stdin      = flag.Bool("i", false, "enable stdin ")
)

func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	return signals
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *mode == CLIENT {
		if *remoteAddr == "" {
			fmt.Printf("client mode need remote addr, use -r")
			return
		}
		err := Client(ctx, *remoteAddr)
		if err != nil {
			fmt.Printf("err:%+v", err)
			return
		}
	}

	if *mode == SERVER {
		if *localAddr == "" || *key == "" {
			fmt.Printf("server mode need local addr and key, use -l and -k")
			return
		}
		Server(ctx, *localAddr, *key)
	}

	if *mode == PROXY {
		if *localAddr == "" || *remoteAddr == "" {
			fmt.Printf("server mode need local addr, use -l")
			return
		}
		Proxy(ctx, *localAddr, *remoteAddr)
	}
	log.Printf("quit signal:%v", <-QuitSignal())
	cancel()
}

func info(conn net.Conn) string {
	return fmt.Sprintf("l:%v<->r:%v", conn.LocalAddr(), conn.RemoteAddr())
}

func Proxy(ctx context.Context, laddr, raddr string) error {
	serverHandle := func(conn net.Conn, _ int) error {
		log.Printf("new conn, l:%v, r:%v\n", conn.LocalAddr(), conn.RemoteAddr())

		nctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		rconn, err := dial.Dial(nctx, raddr,
			dial.WithBackOffer(backoffx.NewDynamicBackoff(time.Second*1, time.Second*10, 1.5)),
			dial.WithKeepAlive(time.Second*20), dial.WithTcpUserTimeout(time.Second*5))
		if err != nil {
			conn.Write([]byte("proxy connect to server fail"))
			return err
		}
		//conn.Write([]byte("you can start a chat now"))
		// 开始转发数据
		go func() {
			_, err := io.Copy(conn, rconn)
			if err != nil {
				fmt.Printf("Failed to forward data from client to server: %v\n", err)
			}
		}()

		_, err = io.Copy(rconn, conn)
		if err != nil {
			fmt.Printf("Failed to forward data from server to client: %v\n", err)
		}

		return err
	}

	s, err := dial.NewServer([]string{laddr}, dial.ServerKeepalive(time.Second*20),
		dial.ServerUserTimeout(time.Second*5), dial.WithHandler(serverHandle))
	if err != nil {
		panic(err)
	}

	s.Start(ctx)
	return nil
}

func Client(ctx context.Context, connectAddr string) error {
	log.Println("client mode")
	reader := bufio.NewReader(os.Stdin)

	hearder := make([]byte, HeaderSize)
	for {
		log.Printf("connecting %s\n", connectAddr)
		conn, err := dial.Dial(ctx, connectAddr,
			dial.WithBackOffer(backoffx.NewDynamicBackoff(time.Second*2, time.Second*30, 2.0)),
			dial.WithKeepAlive(time.Second*20), dial.WithTcpUserTimeout(time.Second*5))
		if err != nil {
			return fmt.Errorf("client dial err:%w", err)
		}
		log.Printf("connecting %s ok\n", connectAddr)
		for {
			fmt.Print("-> ")
			text, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read from stdin err:%w", err)
			}
			// convert CRLF to LF
			//text = strings.Replace(text, "\n", "", -1) //服务器那边是根据\n 作为问题的结束符
			if text == "\n" {
				continue
			}
			wr, err := conn.Write([]byte(text))
			if err != nil {
				log.Println(err)
				break
			}
			_ = wr
			//log.Printf("send request:%d", wr)
			_, err = io.ReadFull(conn, hearder)
			if err != nil {
				log.Println(err)
				break
			}
			payloadLen := binary.BigEndian.Uint16(hearder)
			payload := make([]byte, payloadLen)
			_, err = io.ReadFull(conn, payload)
			if err != nil {
				log.Println(err)
				break
			}
			log.Printf("->answer from chatGPT:\n%s\n", string(payload))
		}
	}
}

func (cc *ChatClient) String() string {
	return fmt.Sprintf("local:%v, remote:%v", cc.conn.LocalAddr(), cc.conn.RemoteAddr())
}

func (cc *ChatClient) Run(ctx context.Context) error {
	defer cc.conn.Close()
	messages := make([]openai.ChatCompletionMessage, 0, DefaultRecordMsgs)

	reader := bufio.NewReader(cc.conn)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("%v, Error reading data:", cc, err.Error())
			return err
		}

		text := string(data)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: text,
		})

		resp, err := cc.aiClient.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:     openai.GPT3Dot5Turbo,
				Messages:  messages,
				MaxTokens: DefaultMaxTokens,
			},
		)
		if err != nil {
			fmt.Println("%v, Error creating completion:", cc, err.Error())
			return err
		}
		content := resp.Choices[0].Message.Content
		if len(messages) == DefaultRecordMsgs {
			//delete first
			messages = append(messages[:0], messages[1:]...)
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: content,
		})
		// 将OpenAI响应转发回客户端
		reply := make([]byte, HeaderSize+len(content))
		binary.BigEndian.PutUint16(reply, uint16(len(content)))
		copy(reply[HeaderSize:], []byte(content))
		// buffer :=bytes.NewBuffer(reply)
		// buffer.W
		_, err = cc.conn.Write(reply)
		if err != nil {
			fmt.Println("%v,Error response data:", cc, err.Error())
			return err
		}
	}
	return nil
}

func Server(ctx context.Context, addr string, key string) {
	log.Println("server mode")
	aiClient := openai.NewClient(key)
	ListModels(ctx, aiClient)

	//chatCh := make(chan chatObject, 1)
	serverHandle := func(conn net.Conn, _ int) error {
		cc := ChatClient{conn: conn, aiClient: aiClient}
		cc.Run(ctx)
		return nil
	}

	s, err := dial.NewServer([]string{addr}, dial.ServerKeepalive(time.Second*20),
		dial.ServerUserTimeout(time.Second*5), dial.WithHandler(serverHandle))
	if err != nil {
		panic(err)
	}
	go s.Start(ctx)

	if *stdin {
		startStdClient(ctx, aiClient)
	}
}

func Balance(ctx context.Context, aiClient *openai.Client) {

	// org, err := aiClient.GetOrganization(ctx)
	// if err != nil {
	// 	// 错误处理
	// }

	// balanceCents := org.BalanceCents
	// balanceDollars := float64(balanceCents) / 100.0

	// fmt.Printf("账户余额：$%.2f\n", balanceDollars)
}

func ListModels(ctx context.Context, aiClient *openai.Client) {
	nctx, ncancel := context.WithTimeout(ctx, time.Second*3)
	modeList, err := aiClient.ListModels(nctx)
	if err != nil {
		log.Printf("get models err:%v", err)
	}
	ncancel()
	log.Printf("models: \n")
	for _, model := range modeList.Models {
		fmt.Printf("%v\n", model.ID)
	}
}

func startStdClient(ctx context.Context, aiClient *openai.Client) {
	messages := make([]openai.ChatCompletionMessage, 0)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("----------Conversation start-----------")
	for {
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)
		if len(text) == 0 {
			continue
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: text,
		})

		resp, err := aiClient.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:     openai.GPT3Dot5Turbo,
				Messages:  messages,
				MaxTokens: DefaultMaxTokens,
			},
		)

		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			continue
		}

		content := resp.Choices[0].Message.Content
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: content,
		})
		fmt.Printf("answer from chatGPT:\n%s\n", content)
		fmt.Printf("------------\n %v ---------\n", resp.Usage)
	}
}

func test() {
	config := openai.DefaultConfig("token")
	proxyUrl, err := url.Parse("http://localhost:{port}")
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}
	config.HTTPClient = &http.Client{
		Transport: transport,
	}

	_ = openai.NewClientWithConfig(config)

	//-------------------------------
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 20,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Lorem ipsum",
			},
		},
		Stream: true,
		//Whether to stream back partial progress. If set, tokens will be sent as data-only
		// server-sent events as they become available, with the stream terminated by a
		// data: [DONE] message.
		//https://platform.openai.com/docs/api-reference/chat/create#chat/create-stream
	}
	c := openai.NewClient("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")
	stream, err := c.CreateChatCompletionStream(ctx, req) //SSE
	if err != nil {
		fmt.Printf("ChatCompletionStream error: %v\n", err)
		return
	}
	defer stream.Close()
}
