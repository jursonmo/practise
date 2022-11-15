package udp

import (
	"context"
	"log"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

func Dial(network, raddr string, data []byte, write int) *net.UDPConn {
	ra, err := net.ResolveUDPAddr(network, raddr)
	if err != nil {
		panic(err)
	}
	uconn, err := net.DialUDP(network, nil, ra)
	if err != nil {
		panic(err)
	}

	for i := 0; i < write; i++ {
		n, err := uconn.Write(data)
		if err != nil {
			panic(err)
		}
		log.Printf("conn local:%s, i:%d, write data len:%d", uconn.LocalAddr().String(), i, n)
	}
	return uconn
}

func ListenReuseport(ctx context.Context, network, addr string, n int) error {
	for i := 0; i < n; i++ {
		//net.ListenPacket(network, addr)
		var lc = net.ListenConfig{
			//listen udp 0.0.0.0:2222: bind: address already in use
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
			return err
		}
		pc := ipv4.NewPacketConn(conn)
		go readLoop(pc, i)
	}
	return nil
}

func readLoop(pc *ipv4.PacketConn, index int) {
	log.Printf("index:%d, listen packet at %s\n", index, pc.LocalAddr().String())
	time.Sleep(time.Second * 2) //等client  发送完数据，查看一次ReadBatch能读多少数据
	log.Printf("listen packet at %s start reading....\n", pc.LocalAddr().String())
	for {
		ms := []ipv4.Message{
			{
				Buffers: [][]byte{make([]byte, 5), make([]byte, 5)}, //一个Buffer装不下完，会放进第二buffer
				//OOB:     ipv4.NewControlMessage(cf),
			},
			{
				Buffers: [][]byte{make([]byte, 12)},
				//OOB:     ipv4.NewControlMessage(cf),
			},
		}
		n, err := pc.ReadBatch(ms, 0)
		if err != nil {
			panic(err)
		}
		log.Printf("index:%d, got n:%d, len(ms):%d\n", index, n, len(ms))

		if n == 0 {
			continue
		}

		for i := 0; i < n; i++ {
			log.Printf("i:%d, from addr:%s, ms.N:%d", i, ms[i].Addr.String(), ms[i].N)
			total := ms[i].N
			for j := 0; j < len((ms[i].Buffers)); j++ {
				log.Printf("ms[%d].Buffers[%d] = %v", i, j, string(ms[i].Buffers[j]))
				total -= len(ms[i].Buffers[j])
				if total <= 0 {
					break
				}
			}
		}
	}
}
