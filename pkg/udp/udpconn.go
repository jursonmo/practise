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
	mux   sync.Mutex
	ln    *Listener
	lconn *net.UDPConn
	pc    *ipv4.PacketConn
	raddr net.Addr
	// rms     []ipv4.Message
	// wms     []ipv4.Message
	// batch   bool
	rxqueue  chan bufferpool.MyBuffer
	rxqueueB chan []byte
	client   bool
	closed   bool
	dead     chan struct{}
}

func NewUDPConn(ln *Listener, lconn *net.UDPConn, raddr net.Addr) *UDPConn {
	uc := &UDPConn{ln: ln, lconn: lconn, raddr: raddr, dead: make(chan struct{}, 1)}
	uc.rxqueue = make(chan bufferpool.MyBuffer, 256)
	uc.rxqueueB = make(chan []byte, 128)
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
		c.ln.deleteConn(c.raddr.String())
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

/* 实现bufio部分接口，让应用层像使用bufio 一样使用UDPConn
func (*bufio.Writer).Write(b []byte)(int, error)
func (*bufio.Writer).Buffered() int
func (*bufio.Writer).Flush() error
*/
type UDPBufioWriter struct {
	c      *UDPConn
	batchs int
	wms    []ipv4.Message
	err    error
}

func NewUDPBufioWriter(c *UDPConn, batchs int) *UDPBufioWriter {
	ub := &UDPBufioWriter{c: c, batchs: batchs}
	ub.wms = make([]ipv4.Message, 0, batchs)

	return ub
}

func (ub *UDPBufioWriter) Write(b []byte) (int, error) {
	if ub.err != nil {
		return 0, ub.err
	}
	ms := ipv4.Message{Buffers: [][]byte{b}}
	ub.wms = append(ub.wms, ms)
	if len(ub.wms) == ub.batchs {
		if err := ub.Flush(); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (ub *UDPBufioWriter) Buffered() int {
	return len(ub.wms)
}

func (ub *UDPBufioWriter) Flush() error {
	if ub.err != nil {
		return ub.err
	}
	wn := len(ub.wms)
	send := 0
	for {
		n, err := ub.c.pc.WriteBatch(ub.wms[send:wn], 0)
		if err != nil {
			ub.err = err
			return err
		}
		send += n
		if send == wn {
			ub.wms = ub.wms[:0]
			return nil
		}
	}
	return nil
}
