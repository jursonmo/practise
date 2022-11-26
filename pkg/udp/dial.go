package udp

import (
	"context"
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

func UdpDial(ctx context.Context, network, laddr, raddr string, opts ...UDPConnOpt) (net.Conn, error) {
	return DialWithOpt(ctx, network, laddr, raddr, opts...)
}

func DialWithOpt(ctx context.Context, network, laddr, raddr string, opts ...UDPConnOpt) (*UDPConn, error) {
	var err error
	la := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	if laddr != "" {
		la, err = net.ResolveUDPAddr(network, laddr)
		if err != nil {
			return nil, err
		}
	}
	ra, err := net.ResolveUDPAddr(network, raddr)
	if err != nil {
		return nil, err
	}

	lconn, err := net.DialUDP(network, la, ra)
	if err != nil {
		return nil, err
	}

	c := NewUDPConn(nil, lconn, ra, opts...)
	// if c.rxhandler != nil {
	// 	go c.ReadBatchLoop(c.rxhandler)
	// }
	return c, nil
}

func (c *UDPConn) ReadBatchLoop(handler func(msg []byte)) error {
	readBatchs := c.readBatchs
	maxBufSize := c.maxBufSize
	pc := c.pc

	rms := make([]ipv4.Message, readBatchs)
	for i := 0; i < len(rms); i++ {
		rms[i] = ipv4.Message{Buffers: [][]byte{make([]byte, maxBufSize)}}
	}
	for {
		n, err := pc.ReadBatch(rms, 0)
		if err != nil {
			c.Close()
			return err
		}
		log.Printf("client ReadBatchLoop got n:%d, len(ms):%d\n", n, len(rms))

		if n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			if handler != nil {
				handler(rms[i].Buffers[0][:rms[i].N])
			}
		}
	}
}

func (c *UDPConn) handlePacket(msg []byte) {
	//分配新的内存对象,并且copy 一次
	b := make([]byte, len(msg))
	copy(b, msg)
	c.PutRxQueue(b)
}
