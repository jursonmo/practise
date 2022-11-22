package udp

import (
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

//readLoop ->handlePacket:分配新的内存对象,并且copy 一次
//为了复用对象,同时减少一次内存copy, 实现 Listener readLoopv2 -> handleBuffer
func (l *Listener) readLoopv2() {
	var err error
	InitPool(l.maxPacketSize)
	rms := make([]ipv4.Message, l.batchs)
	buffers := make([]MyBuffer, l.batchs)
	n := len(rms)
	log.Printf("listener, id:%d, batchs:%d, maxPacketSize:%d, readLoopv2(use MyBuffer)....", l.id, l.batchs, l.maxPacketSize)
	for {
		for i := 0; i < n; i++ {
			b := GetMyBuffer(0)
			buffers[i] = b
			rms[i] = ipv4.Message{Buffers: [][]byte{b.Buffer()}}
		}
		n, err = l.pc.ReadBatch(rms, 0)
		if err != nil {
			l.Close()
			panic(err)
		}
		log.Printf("id:%d, batch got n:%d, len(ms):%d\n", l.id, n, len(rms))

		if n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			buffers[i].AddLen(rms[i].N)
			l.handleBuffer(rms[i].Addr, buffers[i])
		}
	}
}

func (l *Listener) handleBuffer(addr net.Addr, b MyBuffer) {
	uc := l.getUDPConn(addr)
	uc.handleMyBuffer(b)
}

func (c *UDPConn) handleMyBuffer(b MyBuffer) {
	c.PutRxQueue2(b)
}

func (c *UDPConn) PutRxQueue2(b MyBuffer) {
	//非阻塞模式,避免某个UDPConn 的数据没有被处理而阻塞了listener 继续接受数据
	select {
	case c.rxqueue <- b:
	default:
	}
}