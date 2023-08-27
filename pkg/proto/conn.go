package proto

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"time"
)

type ProtoConn struct {
	conn     net.Conn
	isClosed bool
	isServer bool
	authOk   bool

	r              *bufio.Reader
	ReadBufferSize int
	isPacketConn   bool //like udp

	//
	handshaker    func(ctx context.Context, conn net.Conn) error
	handshakeData func() []byte
	msgHandler    ProtoMsgHandle
	pingHandler   func(d []byte) error //invoked when receive ping
	pongHandler   func(d []byte) error //invoked when receive pong

	authReqData func() []byte                 // for client conn, if not nil, means need to send auth request data
	authHandler func(d []byte) ([]byte, bool) //for server conn: it will be invoked when receive request data
}
type ProtoMsgHandle func(pc *ProtoConn, d []byte, t byte) error

var defaultReadBufferSize int = 32 * 1024

func NewProtoConn(c net.Conn, isServer bool, msgHandler ProtoMsgHandle, opts ...ProtoConnOpt) *ProtoConn {
	pc := &ProtoConn{conn: c, isServer: isServer, ReadBufferSize: defaultReadBufferSize}
	pc.r = bufio.NewReaderSize(pc.conn, pc.ReadBufferSize)
	pc.msgHandler = msgHandler
	pc.SetPingHandler(pc.WritePong) //默认会设置回应Pong 消息，payload 不变
	pc.SetPongHandler(nil)
	pc.SetAuthHandler(nil) //默认是设置为nil, 即不需要验证，authOk 初始值为true, 如果需要验证，WithAuthHandler设置

	for _, opt := range opts {
		opt(pc)
	}
	return pc
}

func (pc *ProtoConn) String() string {
	return fmt.Sprintf("%v<->%v", pc.LocalAddr(), pc.RemoteAddr())
}
func (pc *ProtoConn) Conn() net.Conn {
	return pc.conn
}
func (pc *ProtoConn) LocalAddr() net.Addr {
	return pc.conn.LocalAddr()
}
func (pc *ProtoConn) RemoteAddr() net.Addr {
	return pc.conn.RemoteAddr()
}
func (c *ProtoConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *ProtoConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (pc *ProtoConn) Close() {
	pc.conn.Close()
	pc.isClosed = true

}

type ProtoConnOpt func(*ProtoConn)

func WithAuthHandler(f func(d []byte) ([]byte, bool)) ProtoConnOpt {
	return func(pc *ProtoConn) {
		pc.SetAuthHandler(f)
	}
}

func WithAuthReqData(f func() []byte) ProtoConnOpt {
	return func(pc *ProtoConn) {
		pc.authReqData = f
	}
}

var DefaultHandShakeTimeout = time.Second * 2

//握手一般用于udp 这种基于报文的协议，握手成功表示连接可以收发数据了， tcp 默认就有三次握手，所以可以不用设置握手，用authHandler
//这种方式设置握手的时所用的数据，只需在client 和 server 端设置相同的数据就能握手成功, 握手过程的收发操作底层默认实现。
func WithHandShakeData(f func() []byte) ProtoConnOpt {
	return func(pc *ProtoConn) {
		pc.handshakeData = f
	}
}

//如果用这种方式，自己实现conn 的收发操作，以及对比数据，比如 DefaultHandShake 的做法
func WithHandShake(f func(ctx context.Context, conn net.Conn) error) ProtoConnOpt {
	return func(pc *ProtoConn) {
		pc.handshaker = f
	}
}

//
func DefaultHandShake(ctx context.Context, conn net.Conn) error {
	handshakeToken := []byte("hello")
	_, err := conn.Write(handshakeToken)
	if err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Second)
	}
	conn.SetReadDeadline(deadline)

	buf := make([]byte, len(handshakeToken))
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return err
	}

	conn.SetReadDeadline(time.Time{})
	if !reflect.DeepEqual(buf, handshakeToken) {
		return errors.New("handshake fail")
	}
	return err
}

func (pc *ProtoConn) SetMsgHandler(h ProtoMsgHandle) {
	pc.msgHandler = h
}

func (pc *ProtoConn) PingHandler() func(data []byte) error {
	return pc.pingHandler
}

func (pc *ProtoConn) SetPingHandler(h func(data []byte) error) {
	pc.pingHandler = h
}

func (pc *ProtoConn) WritePing(d []byte) error {
	pingPkg, err := NewPingPkg(d)
	if err != nil {
		return err
	}
	_, err = pc.conn.Write(pingPkg.Bytes())
	return err
}

func (pc *ProtoConn) WritePong(d []byte) error {
	pongPkg, err := NewPongPkg(d)
	if err != nil {
		return err
	}
	_, err = pc.conn.Write(pongPkg.Bytes())
	return err
}

func (pc *ProtoConn) PongHandler() func(data []byte) error {
	return pc.pongHandler
}

func (pc *ProtoConn) SetPongHandler(h func(data []byte) error) {
	if h == nil {
		h = func([]byte) error { return nil }
	}
	pc.pongHandler = h
}

func (pc *ProtoConn) SetAuthHandler(h func(data []byte) ([]byte, bool)) {
	//如果设置成nil,表示不需要auth, 直接设置成authOk=true
	if h == nil {
		pc.authOk = true
		pc.authHandler = nil
		return
	}
	pc.authOk = false
	pc.authHandler = h
}

func (pc *ProtoConn) SendAuthReq(d []byte) error {
	authreq, err := NewAuthReqPkg(d)
	if err != nil {
		return err
	}
	fmt.Printf("authreq:%v\n", authreq)
	_, err = pc.conn.Write(authreq.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (pc *ProtoConn) WriteCloseMsg(code int, msg string) (int, error) {
	data, err := json.Marshal(CloseCmdPayLoad{Code: code, Msg: msg})
	if err != nil {
		return 0, err
	}
	pkg, err := EncodeCmdPkg(CloseCmd, data)
	if err != nil {
		return 0, err
	}
	return pc.conn.Write(pkg.Bytes())
}

var ErrUnauth = errors.New("unauth")

//实际上是发送 Msg 类型的数据
//发送用户数据前，检查auth 是否ok
func (pc *ProtoConn) Write(d []byte) (int, error) {
	if !pc.authOk {
		return 0, ErrUnauth
	}
	pkg, err := EncodePkg(d, Msg, 0)
	if err != nil {
		return 0, err
	}
	return pc.conn.Write(pkg.Bytes())
}

func (pc *ProtoConn) clientHandshake(ctx context.Context) error {
	d := pc.handshakeData()
	pingPkg, err := NewPingPkg(d)
	if err != nil {
		return err
	}
	_, err = pc.conn.Write(pingPkg.Bytes())
	if err != nil {
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(DefaultHandShakeTimeout)
	}
	pc.conn.SetReadDeadline(deadline)
	defer pc.conn.SetReadDeadline(time.Time{})

	handshake := NewProtoPkg()
	err = handshake.Decode(pc.r)
	if err != nil {
		return err
	}
	fmt.Printf("reply handshake:%v\n", handshake)
	if !reflect.DeepEqual(handshake.Payload, d) {
		fmt.Printf("receive handshake payload:%s, handshakeData:%s\n", string(handshake.Payload), string(d))
		return errors.New("handshake fail")
	}
	return nil
}

func (pc *ProtoConn) serverHandshake(ctx context.Context) error {
	d := pc.handshakeData()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(DefaultHandShakeTimeout)
	}
	pc.conn.SetReadDeadline(deadline)
	defer pc.conn.SetReadDeadline(time.Time{})

	handshake := NewProtoPkg()
	err := handshake.Decode(pc.r)
	if err != nil {
		return err
	}
	fmt.Printf("receive handshake request:%v\n", handshake)
	if !reflect.DeepEqual(handshake.Payload, d) {
		fmt.Printf("receive handshake payload:%s, don't match handshakeData:%s\n", string(handshake.Payload), string(d))
		return errors.New("handshake fail")
	}
	//if this is ping ,response pong
	if handshake.PkgType() == Ping {
		if pc.pingHandler != nil {
			return pc.pingHandler(handshake.Payload)
		}
	}
	return nil
}

func (pc *ProtoConn) Handshake(ctx context.Context) error {
	if pc.handshaker != nil {
		return pc.handshaker(ctx, pc.conn)
	}

	//默认用ping pong 来表示握手
	if pc.handshakeData != nil {
		if pc.isServer {
			return pc.serverHandshake(ctx)
		}
		return pc.clientHandshake(ctx)
	}
	return nil
}

var DefaultAuthTimeout = time.Second * 2

// client send auth request
func (pc *ProtoConn) clientAuth(ctx context.Context) error {
	if pc.authReqData == nil {
		return nil
	}
	d := pc.authReqData()
	err := pc.SendAuthReq(d)
	if err != nil {
		return err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(DefaultAuthTimeout)
	}
	pc.conn.SetReadDeadline(deadline)
	defer pc.conn.SetReadDeadline(time.Time{})

	authresp := NewProtoPkg()
	err = authresp.Decode(pc.r)
	if err != nil {
		return err
	}
	fmt.Printf("auth reply :%v\n", authresp)
	if len(authresp.options) == 0 {
		return errors.New("get response fail")
	}
	opt := authresp.options[0]
	fmt.Printf("authresp.options:%v\n", &opt)
	if opt.T != AuthOk {
		return errors.New(string(opt.V))
	}
	return nil
}

//server handler auth request and response
func (pc *ProtoConn) serverAuth(ctx context.Context) error {
	if pc.authHandler == nil {
		return nil
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(DefaultAuthTimeout)
	}
	pc.conn.SetReadDeadline(deadline)
	defer pc.conn.SetReadDeadline(time.Time{})

	authReqPkg := NewProtoPkg()
	err := authReqPkg.Decode(pc.r)
	if err != nil {
		return err
	}
	if authReqPkg.PkgType() != Auth {
		return errors.New("isn't auth packet")
	}
	if len(authReqPkg.options) == 0 {
		return errors.New("get response fail")
	}
	opt := authReqPkg.options[0]
	authRes, authOk := pc.authHandler(opt.V)
	authresPkg, err := NewAuthRespPkg(authRes, authOk)
	if err != nil {
		return err
	}
	_, err = pc.conn.Write(authresPkg.Bytes())
	if err != nil {
		return err
	}
	pc.authOk = authOk
	return nil
}

func (pc *ProtoConn) Auth(ctx context.Context) error {
	if pc.isServer {
		pc.serverAuth(ctx)
	} else {
		return pc.clientAuth(ctx)
	}
	return nil
}

//Run, read loop
func (pc *ProtoConn) Run(ctx context.Context) error {
	defer pc.Close()

	var err error
	defer func() {
		log.Printf("ProtoConn Run task quit, err:%v", err)
	}()

	for {
		pkg := NewProtoPkg()
		err = pkg.Decode(pc.r)
		if err != nil {
			return err
		}
		t := pkg.PkgType()
		switch t {
		case Msg:
			if !pc.authOk {
				log.Printf("haven't auth ok")
				continue
			}
			if pc.msgHandler == nil {
				log.Printf("haven't set raw msg Handler ?")
				continue
			}
			err = pc.msgHandler(pc, pkg.Payload, byte(pkg.PayloadType()))
			if err != nil {
				return fmt.Errorf("msgHandler err:%w", err)
			}
		case Ping:
			if pc.pingHandler == nil {
				continue
			}
			//默认是echo, 即回应pong,数据是原来的数据
			err = pc.pingHandler(pkg.Payload)
			if err != nil {
				return fmt.Errorf("pingHandler err:%w", err)
			}
		case Pong:
			if pc.pongHandler == nil {
				continue
			}
			err = pc.pongHandler(pkg.Payload)
			if err != nil {
				return fmt.Errorf("pongHandler err:%w", err)
			}
		case Auth:
			//如果服务器设置authHandler, 就会在serverAuth 处理auth 请求。
			//如果服务器没有设置authHandler, 而client 要求auth, 就会走到这里，
			if pc.authHandler == nil {
				//没有设置authHandler默认认证Ok,返回认证OK信息
				authresp, err := NewAuthRespPkg(nil, true)
				if err != nil {
					return err
				}
				_, err = pc.Write(authresp.Bytes())
				if err != nil {
					return err
				}
				continue
			}
		}
	}
}

//初始化，握手或者验证，或者两个都做，
func (pc *ProtoConn) Init(ctx context.Context) error {
	//是否配置handshaker 或者 handshakeData
	err := pc.Handshake(ctx)
	if err != nil {
		return err
	}
	//auth:
	err = pc.Auth(ctx)
	if err != nil {
		return err
	}
	return nil
}
