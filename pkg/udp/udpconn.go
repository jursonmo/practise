package udp

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	bufferpool "github.com/jursonmo/practise/pkg/bufferPool"
	"golang.org/x/net/ipv4"
)

var ErrConnClosed = errors.New("Conn Closed")

type UDPConn struct {
	mux    sync.Mutex
	ln     *Listener
	client bool //client 表示lconn is conneted(绑定了目的地址), 即可以直接用Write，不需要WriteTo
	lconn  *net.UDPConn
	pc     *ipv4.PacketConn
	raddr  *net.UDPAddr
	// rms     []ipv4.Message
	// wms     []ipv4.Message
	// batch   bool
	rxqueue    chan bufferpool.MyBuffer
	rxqueueB   chan []byte
	rxhandler  func([]byte)
	rxqueuelen int
	readBatchs int
	maxBufSize int

	closed bool
	dead   chan struct{}
}

type UDPConnOpt func(*UDPConn)

func WithRxQueueLen(n int) UDPConnOpt {
	return func(u *UDPConn) {
		u.rxqueuelen = n
	}
}

func WithReadBatchs(n int) UDPConnOpt {
	return func(u *UDPConn) {
		u.readBatchs = n
	}
}
func WithMaxPacketSize(n int) UDPConnOpt {
	return func(u *UDPConn) {
		u.maxBufSize = n
	}
}

func WithRxHandler(h func([]byte)) UDPConnOpt {
	return func(u *UDPConn) {
		u.rxhandler = h
	}
}

func NewUDPConn(ln *Listener, lconn *net.UDPConn, raddr *net.UDPAddr, opts ...UDPConnOpt) *UDPConn {
	uc := &UDPConn{ln: ln, lconn: lconn, raddr: raddr, dead: make(chan struct{}, 1),
		rxqueuelen: 256,
		readBatchs: 8,
		maxBufSize: 1600,
	}
	uc.rxhandler = uc.handlePacket
	for _, opt := range opts {
		opt(uc)
	}
	uc.rxqueue = make(chan bufferpool.MyBuffer, uc.rxqueuelen)
	uc.rxqueueB = make(chan []byte, uc.rxqueuelen)
	if uc.ln == nil {
		uc.client = true
	}
	uc.pc = ipv4.NewPacketConn(lconn)
	return uc
}

func (c *UDPConn) PutRxQueue(data []byte) {
	c.rxqueueB <- data
}

func (c *UDPConn) Close() error {
	c.mux.Lock()
	if c.closed == true {
		c.mux.Unlock()
		return nil
	}
	c.closed = true
	c.mux.Unlock()

	close(c.dead)
	if c.ln != nil {
		if key, ok := udpAddrTrans(c.raddr); ok {
			c.ln.deleteConn(key)
		}
	}
	if c.client && c.lconn != nil {
		c.lconn.Close()
		log.Printf("client:%v, %s<->%s, raddr:%s close over\n", c.client, c.LocalAddr().String(), c.RemoteAddr().String())
	}
	return nil
}

func (c *UDPConn) LocalAddr() net.Addr {
	return c.lconn.LocalAddr()
}
func (c *UDPConn) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *UDPConn) Read(buf []byte) (n int, err error) {
	select {
	case b := <-c.rxqueueB:
		n = copy(buf, b)
		//todo
		return
	case b := <-c.rxqueue:
		n, err = b.Read(buf)
		//todo
		return
	case <-c.dead:
		return 0, ErrConnClosed
	}
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	if c.client {
		return c.lconn.Write(b)
	}
	return c.lconn.WriteTo(b, c.raddr)
}

func (c *UDPConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *UDPConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *UDPConn) SetWriteDeadline(t time.Time) error {
	return nil
}
