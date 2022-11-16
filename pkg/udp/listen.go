package udp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

var ErrLnClosed = errors.New("udp listener closed")

type LnCfgOptions func(*ListenConfig)

func WithReuseport(b bool) LnCfgOptions {
	return func(lc *ListenConfig) {
		lc.reuseport = true
	}
}

func WithListenerNum(n int) LnCfgOptions {
	return func(lc *ListenConfig) {
		lc.listenerNum = n
	}
}

type ListenConfig struct {
	network string
	addr    string
	//raddr     string
	reuseport   bool
	listenerNum int
}

type UdpListen struct {
	sync.Mutex
	ctx       context.Context
	listeners []*Listener
	laddr     *net.UDPAddr
	accept    chan net.Conn
	dead      chan struct{}
	closed    bool
	cfg       ListenConfig
}

func NewUdpListen(ctx context.Context, network, addr string, opts ...LnCfgOptions) (*UdpListen, error) {
	cfg := ListenConfig{network: network, addr: addr}
	for _, opt := range opts {
		opt(&cfg)
	}
	err := cfg.Tidy()
	if err != nil {
		return nil, err
	}

	ln := &UdpListen{ctx: ctx, cfg: cfg}
	err = ln.Start()
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func (ln *UdpListen) Start() error {
	cfg := ln.cfg
	laddr, err := net.ResolveUDPAddr(cfg.network, cfg.addr)
	if err != nil {
		return err
	}
	ln.laddr = laddr
	ln.listeners = make([]*Listener, cfg.listenerNum)
	for i := 0; i < cfg.listenerNum; i++ {
		l, err := NewListener(ln.ctx, cfg.network, cfg.addr, WithId(i))
		if err != nil {
			log.Println(err)
			continue
		}
		ln.listeners[i] = l

	}
	ln.Listen()
	return nil
}

func (cfg *ListenConfig) Tidy() error {
	if cfg.network == "" || cfg.addr == "" {
		return fmt.Errorf("network or addr is empty")
	}

	if !cfg.reuseport {
		cfg.listenerNum = 1
	}
	if cfg.reuseport && cfg.listenerNum == 0 {
		cfg.listenerNum = runtime.GOMAXPROCS(0)
	}
	return nil
}

func (ln *UdpListen) Listen() {
	for _, l := range ln.listeners {
		if l == nil {
			continue
		}
		go func(l *Listener) {
			log.Printf("%v listenning....", l)
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				ln.accept <- conn
			}
		}(l)
	}
}

// 实现 net.Listener 接口 Accept()、Addr() 、Close()
func (l *UdpListen) Accept() (net.Conn, error) {
	for {
		//check if dead first
		select {
		case <-l.dead:
			return nil, ErrLnClosed
		default:
		}

		select {
		case <-l.dead:
			return nil, ErrLnClosed
		case conn, ok := <-l.accept:
			if !ok {
				return nil, ErrLnClosed
			}
			return conn, nil
		}
	}
}

func (l *UdpListen) Addr() net.Addr {
	return l.laddr
}

func (l *UdpListen) Close() error {
	l.Lock()
	if l.closed {
		l.Unlock()
		return ErrLnClosed
	}
	l.closed = true
	l.Unlock()

	close(l.dead)
	close(l.accept)

	for _, listener := range l.listeners {
		if listener == nil {
			continue
		}
		listener.Close()
	}
	return nil
}

type Listener struct {
	sync.Mutex
	id    int
	lconn *net.UDPConn
	pc    *ipv4.PacketConn
	//ln      net.Listener
	clients sync.Map
	accept  chan *UDPConn
	dead    chan struct{}
	closed  bool
}
type ListenerOpt func(*Listener)

func WithId(id int) ListenerOpt {
	return func(l *Listener) {
		l.id = id
	}
}
func NewListener(ctx context.Context, network, addr string, opts ...ListenerOpt) (*Listener, error) {
	l := &Listener{}
	for _, opt := range opts {
		opt(l)
	}
	l.accept = make(chan *UDPConn, 128)

	var lc = net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			}); err != nil {
				return err
			}
			return opErr
		},
	}
	conn, err := lc.ListenPacket(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	l.lconn = conn.(*net.UDPConn)
	l.pc = ipv4.NewPacketConn(conn)
	l.readLoop()
	return l, nil
}

func (l *Listener) readLoop() {
	readBatchs := 8
	rms := make([]ipv4.Message, readBatchs)
	for i := 0; i < 8; i++ {
		rms[i] = ipv4.Message{Buffers: [][]byte{make([]byte, 1600)}}
	}
	for {
		n, err := l.pc.ReadBatch(rms, 0)
		if err != nil {
			l.Close()
			panic(err)
		}
		log.Printf("id:%d, got n:%d, len(ms):%d\n", l.id, n, len(rms))

		if n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			l.handlePacket(rms[i].Addr, rms[i].Buffers[0][:rms[i].N])
		}
	}
}

func (l *Listener) handlePacket(addr net.Addr, data []byte) {
	var uc *UDPConn
	raddr := addr.String()
	v, ok := l.clients.Load(raddr)
	if !ok {
		//new udpConn
		uc = NewUDPConn(l, l.lconn, addr)
		l.accept <- uc
	} else {
		uc = v.(*UDPConn)
	}
	uc.PutRxQueue(data)
}

func (l *Listener) deleteConn(key string) {
	log.Printf("id:%d, del: %s, local:%s, remote: %s", l.id, l.LocalAddr().Network(), l.LocalAddr().String(), key)
	l.clients.Delete(key)
}

func (l *Listener) LocalAddr() net.Addr {
	return l.lconn.LocalAddr()
}

// 实现 net.Listener 接口 Accept()、Addr() 、Close()
func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.dead:
			return nil, ErrLnClosed
		case c, ok := <-l.accept:
			if !ok {
				return nil, ErrLnClosed
			}
			return c, nil
		}
	}
}

func (l *Listener) Addr() net.Addr {
	return l.LocalAddr()
}

func (l *Listener) Close() error {
	l.Lock()
	if l.closed {
		return ErrLnClosed
	}
	l.closed = true
	l.Unlock()
	log.Printf("%v closing....", l)
	defer log.Printf("%v over", l)
	close(l.dead)
	close(l.accept)

	return l.lconn.Close()
}

func (l *Listener) String() string {
	return fmt.Sprintf("listener, id:%d, local:%s", l.id, l.LocalAddr().String())
}