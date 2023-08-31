package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/jursonmo/practise/pkg/dial"
	"github.com/jursonmo/practise/pkg/heartbeat"
	"github.com/jursonmo/practise/pkg/proto"
	"github.com/jursonmo/practise/pkg/proto/session"
	"golang.org/x/sync/errgroup"
)

var ErrInvalidData = errors.New("invalid data")

type Client struct {
	sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	eg        *errgroup.Group
	name      string
	endpoints []*url.URL
	conn      net.Conn
	closed    bool

	//handlers
	onDialFail func(error)
	onConnect  func(session.Sessioner) error
	onStop     func(string)

	pc      *proto.ProtoConn
	routers *session.RouterRegister
	// isServer bool
	// authOk   bool

	// r              *bufio.Reader
	// ReadBufferSize int
	// isPacketConn   bool //like udp

	// handshaker    func(ctx context.Context, conn net.Conn) error
	// handshakeData func() []byte
	// msgHandler    func(c *Client, d []byte) error
	// pingHandler   func(d []byte) error //invoked when receive ping
	// pongHandler   func(d []byte) error //invoked when receive pong

	// authReqData func() []byte                 // for client conn, if not nil, means need to send auth request data
	// authHandler func(d []byte) ([]byte, bool) //for server conn: it will be invoked when receive request data

}

//实现Sessioner 接口
func (c *Client) Name() string {
	return c.name
}

func (c *Client) SessionID() string {
	if c.conn != nil {
		return fmt.Sprintf("%v->%v", c.conn.LocalAddr(), c.conn.RemoteAddr())
	}
	return "non session"
}

func (c *Client) UnderlayConn() net.Conn {
	return c.conn
}

func (c *Client) Endpoints() []*url.URL {
	return c.endpoints
}

func (c *Client) String() string {
	return fmt.Sprintf("name:%v, id:%v", c.name, c.SessionID())
}

func NewClient(endpoints []string, opts ...Option) (*Client, error) {
	c := &Client{routers: session.NewRouterRegister()}
	for _, endpoint := range endpoints {
		url, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		c.endpoints = append(c.endpoints, url)
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	var egctx context.Context
	go func() error {
		for {
			if err := c.ctx.Err(); err != nil {
				return err
			}
			e := c.endpoints[0]
			addr := e.Scheme + "://" + e.Host
			conn, err := dial.Dial(c.ctx, addr,
				dial.WithBackOffer(backoffx.NewDynamicBackoff(time.Second*2, time.Second*30, 2.0)),
				dial.WithKeepAlive(time.Second*5), dial.WithTcpUserTimeout(time.Second*5), dial.WithDialFailFunc(c.onDialFail))
			if err != nil {
				log.Println("dial stoped, err:", err)
				//dial 会一直在后台拨号，知道ctx 被cancel, 所以dial返回err 后，就需要继续dial 来，直接返回
				return err
			}

			c.conn = conn
			c.pc = proto.NewProtoConn(conn, false, proto.ProtoMsgHandle(c.msgHandle))
			//如果设置了SetMsgHandlerv2, 那么c.msgHandle 就不起作用
			c.pc.SetMsgHandlerv2(proto.ProtoMsgHandlev2(c.msgHandlev2))

			err = c.pc.Init(c.ctx)
			if err != nil {
				log.Println("pc.Init err:", err)
				continue
			}
			s := NewSession(c, c.pc)
			if c.onConnect != nil {
				//c.onConnect(c)
				go c.onConnect(s)
			}

			c.eg, egctx = errgroup.WithContext(c.ctx) //要用c.ctx, 这样c.cancel 才能 取消egctx
			c.eg.Go(func() error {
				err := c.pc.Run(egctx)
				log.Println(err)
				return err
			})

			//注册heartbeat 处理请求回调，默认就是回应原始数据,类型是HeartBeatRespId
			c.addRouter(uint16(session.HeartBeatReqId),
				session.HandleFunc(func(s session.Sessioner, msgid uint16, d []byte) {
					log.Printf("receive hb request:%s", string(d))
					err := s.WriteMsg(session.HeartBeatRespId, d)
					if err != nil {
						log.Println("receive hb request, and send hb response err:", err)
					}
				}))

			//发送心跳
			hbsend := func(req heartbeat.HbPkg) error {
				buf := bytes.NewBuffer(make([]byte, 0, 128))
				//binary.Write(buf, binary.BigEndian, uint16(session.HeartBeatReqId))
				encoder := json.NewEncoder(buf)
				err := encoder.Encode(&req)
				if err != nil {
					return err
				}
				log.Printf("send heartbeat requet len:%d, data:%s", len(buf.Bytes()), buf.String())
				//_, err = c.pc.Write(buf.Bytes())

				_, err = c.pc.WriteWithId(session.HeartBeatReqId, buf.Bytes())
				return err
			}

			//todo抽象出：heartbeater interface{}
			heartbeater := heartbeat.NewHeartbeat(c.name,
				heartbeat.DefautConfig, hbsend)

			//注册心跳回应处理
			c.addRouter(uint16(session.HeartBeatRespId), session.HandleFunc(func(s session.Sessioner, id uint16, d []byte) {
				hb := heartbeat.HbPkg{}
				err := json.Unmarshal(d, &hb)
				if err != nil {
					log.Printf("handle hb response err:%v", err)
					return
				}
				log.Printf("receive hb response:%+v\n", hb)
				//return //模拟心跳收不到的情况
				heartbeater.PutResponse(hb)
			}))

			c.eg.Go(func() error {
				err := heartbeater.Start(egctx)
				log.Printf("heartbeat quit, err:%v", err)
				return err
			})

			c.eg.Go(func() error {
				<-egctx.Done()
				c.pc.Close() //make pc.Run() quit
				return egctx.Err()
			})
			c.eg.Wait()
		}
	}()
	return nil
}

func (c *Client) AddRouter(msgid uint16, r session.Router) error {
	//业务数据id 从10开始，0-9 预留给了心跳报文
	if err := session.CheckMsgId(msgid); err != nil {
		return err
	}
	return c.addRouter(msgid, r)
}

func (c *Client) addRouter(msgid uint16, r session.Router) error {
	return c.routers.AddRouter(msgid, r)
}

func (c *Client) GetRouter(msgid uint16) session.Router {
	return c.routers.GetRouter(msgid)
}

func (c *Client) msgHandle(pc *proto.ProtoConn, d []byte, t byte) error {
	if len(d) < 3 {
		return ErrInvalidData
	}
	msgid := binary.BigEndian.Uint16(d)
	r := c.GetRouter(msgid)
	if r == nil {
		return nil
	}
	r.Handle(c, msgid, d[2:])
	return nil
}

func (c *Client) msgHandlev2(pc *proto.ProtoConn, pkg proto.Pkger) error {
	msgid, ok := pkg.MsgId()
	if !ok {
		log.Println("msgHandlev2 can't get msgid")
		return nil
	}
	r := c.GetRouter(msgid)
	if r == nil {
		return nil
	}
	r.Handle(c, msgid, pkg.Paylaod())
	return nil
}

func (c *Client) WriteMsg(msgid uint16, d []byte) error {
	// 这里需要make 一个大的内存对象，还需要copy一次
	// buf := make([]byte, len(d)+2)
	// binary.BigEndian.PutUint16(buf, msgid)
	// copy(buf[2:], d)
	// _, err := c.pc.Write(buf)
	// return err

	return c.WriteMsgv2(msgid, d)
}

func (c *Client) WriteMsgv2(msgid uint16, d []byte) error {
	_, err := c.pc.WriteWithId(msgid, d)
	return err
}

func (c *Client) Stop(ctx context.Context) error {
	c.Lock()
	defer c.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	log.Printf("client:%v Stopping\n", c)
	if c.cancel != nil {
		c.cancel()
	}
	c.pc.Close()

	if c.onStop != nil {
		c.onStop(c.Name())
	}
	return nil
}
