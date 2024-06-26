package udp

import (
	"errors"
	"log"

	"golang.org/x/net/ipv4"
)

var ErrTooBig = errors.New("bigger than Buffer MaxSize")

//var ErrTxQueueFull = errors.New("Err txqueueu is full")

//use listener write batch, 把data 转换成MyBuffer, 然后放到tx队列里
func (c *UDPConn) WriteWithBatch(data []byte) (n int, err error) {
	b := GetMyBuffer(len(data))
	if b == nil {
		//data too bigger?
		return 0, ErrTooBig
	}
	n, err = b.Write(data)
	if err != nil {
		panic(err)
		//return
	}

	if c.ln != nil {
		b.SetAddr(c.raddr)
		err = c.ln.PutTxQueue(b)
	} else {
		err = c.PutTxQueue(b)
	}
	if err != nil {
		return 0, err
	}
	return
}

//返回的error 应该实现net.Error temporary(), 这样上层Write可以认为Eagain,再次调用Write
func (l *Listener) PutTxQueue(b MyBuffer) error {
	select {
	case l.txqueue <- b:
	default:
		Release(b)
		return ErrTxQueueFull
	}
	return nil
}

func (l *Listener) WriteBatchAble() bool {
	return l.writeBatchAble
}

func (l *Listener) writeBatchLoop() {
	bw, _ := NewPCBioWriter(l.pc, l.batchs)
	l.writeBatchAble = true
	defer func() { l.writeBatchAble = false }()
	defer log.Printf("id:%d, listener %v, writeBatchLoop quit", l.id, l.pc.LocalAddr())

	bw.WriteBatchLoop(l.txqueue)
	/*
		var err error
		for b := range l.txqueue {
			//为什么不把"data[]byte 转换成Mybuffer" 放在WriteWithBatch()实现,而不放在这里实现呢,
			//如果放在这里实现，PCBufioWriter 就可以实现bufioer 接口了
			//因为上层调用write(data []byte)后，默认是data 被发送出去了,并认为可以重用这个data的
			//如果把[]byte 放在txqueue 队列里, 那么这个data []byte 在生成MyBuffer前，可能被修改了.
			_, err = bw.Write(b)
			if err != nil {
				log.Println(err)
				return
			}
			if len(l.txqueue) == 0 && bw.Buffered() > 0 {
				err = bw.Flush()
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	*/
}

type writeBatchMsg struct {
	offset  int //send offset
	wms     []ipv4.Message
	buffers []MyBuffer
}

//PacketConnBufioWriter
type PCBufioWriter struct {
	pc     *ipv4.PacketConn
	batchs int
	writeBatchMsg
	err error
}

func NewPCBioWriter(pc *ipv4.PacketConn, batchs int) (*PCBufioWriter, error) {
	if batchs == 0 {
		batchs = defaultBatchs
	}
	bw := &PCBufioWriter{pc: pc, batchs: batchs}

	// bw.wms = make([]ipv4.Message, 0, batchs)
	// bw.buffers = make([]MyBuffer, 0, batchs)
	bw.writeBatchMsg.init(batchs)
	return bw, nil
}

//由于*PCBufioWriter 只能实现Write(b MyBuffer)，而不是Write([]byte) (n int, err error)
//所以*PCBufioWriter 并没有实现 Bufioer 接口。
/*
func (bw *PCBufioWriter) Write(b MyBuffer) (n int, err error) {
	if bw.err != nil {
		return 0, bw.err
	}

	ms := ipv4.Message{Buffers: [][]byte{b.Bytes()}, Addr: b.GetAddr()}
	bw.wms = append(bw.wms, ms)
	bw.buffers = append(bw.buffers, b)
	if len(bw.wms) == bw.batchs {
		if err := bw.Flush(); err != nil {
			return 0, err
		}
	}
	return len(b.Bytes()), nil
}

func (bw *PCBufioWriter) Buffered() int {
	return len(bw.wms)
}

func (bw *PCBufioWriter) Flush() error {
	log.Printf("listener %v, flushing %d packet....", bw.pc.LocalAddr(), len(bw.wms))
	if bw.err != nil {
		return bw.err
	}
	wn := len(bw.wms)
	send := 0
	for {
		n, err := bw.pc.WriteBatch(bw.wms[send:wn], 0)
		if err != nil {
			bw.err = err
			return err
		}
		bw.ReleaseMyBuffer(send, send+n)
		send += n
		if send == wn {
			bw.wms = bw.wms[:0]
			bw.buffers = bw.buffers[:0]
			return nil
		}
	}
}

func (bw *PCBufioWriter) ReleaseMyBuffer(from, to int) {
	for i := from; i < to; i++ {
		Release(bw.buffers[i])
	}
}
*/

//由于*PCBufioWriter 只能实现Write(b MyBuffer)，而不是Write([]byte) (n int, err error)
//所以*PCBufioWriter 并没有实现 Bufioer 接口。
func (bw *PCBufioWriter) Write(b MyBuffer) (n int, err error) {
	if bw.err != nil {
		return 0, bw.err
	}
	if flush := bw.addMsg(b); flush {
		if err := bw.Flush(); err != nil {
			return 0, err
		}
	}
	return len(b.Bytes()), nil
}

func (bw *PCBufioWriter) Buffered() int {
	return bw.buffered()
}

//not return until flush all msg; 直到发送完缓存中的所有数据才返回
func (bw *PCBufioWriter) Flush() error {
	log.Printf("local %v, flushing %d packet....", bw.pc.LocalAddr(), len(bw.wms))
	if bw.err != nil {
		return bw.err
	}

	for {
		msgs := bw.msgBuffered()
		if len(msgs) == 0 {
			return nil
		}
		n, err := bw.pc.WriteBatch(msgs, 0) //如果不是linux 平台，会报错：sendmsg invaild parameter
		if err != nil {
			bw.err = err
			return err
		}
		bw.commit(n)
	}
}

func (w *writeBatchMsg) init(capability int) {
	w.offset = 0
	w.wms = make([]ipv4.Message, 0, capability)
	w.buffers = make([]MyBuffer, 0, capability)
}

func (w *writeBatchMsg) buffered() int {
	return len(w.wms) - w.offset
	//return len(w.msgBuffered())
}

func (w *writeBatchMsg) addMsg(b MyBuffer) (flush bool) {
	ms := ipv4.Message{Buffers: [][]byte{b.Bytes()}, Addr: b.GetAddr()}
	w.wms = append(w.wms, ms)
	w.buffers = append(w.buffers, b)
	if len(w.wms) == cap(w.wms) {
		return true
	}
	return false
}

//获取需要发送的消息
func (w *writeBatchMsg) msgBuffered() []ipv4.Message {
	return w.wms[w.offset:]
}

func (w *writeBatchMsg) commit(sended int) {
	if sended == 0 {
		return
	}
	//已经发送的消息，可以释放
	for i := w.offset; i < w.offset+sended; i++ {
		w.wms[i].Buffers[0] = nil //set nil for gc
		Release(w.buffers[i])     //release buffer to pool
	}

	//update offset
	w.offset += sended

	//just check
	if w.offset > len(w.wms) {
		log.Panicf("w.offset:%d, > len(w.wms):%d", w.offset, len(w.wms))
	}

	//已经全部发完了，重置
	if w.offset == len(w.wms) {
		//log.Printf("--------- w.offset:%d, 已经全部发完了，重置\n", w.offset) //test ok
		w.offset = 0
		w.wms = w.wms[:0]
		w.buffers = w.buffers[:0]
	}
}

func (bw *PCBufioWriter) WriteBatchLoop(fromCh chan MyBuffer) {
	var err error
	for b := range fromCh {
		//为什么不把"data[]byte 转换成Mybuffer" 放在WriteWithBatch()实现,而不放在这里实现呢,
		//如果放在这里实现，PCBufioWriter 就可以实现bufioer 接口了
		//因为上层调用write(data []byte)后，默认是data 被发送出去了,并认为可以重用这个data的
		//如果把[]byte 放在txqueue 队列里, 那么这个data []byte 在生成MyBuffer前，可能被修改了.
		_, err = bw.Write(b)
		if err != nil {
			log.Println(err)
			return
		}
		if len(fromCh) == 0 && bw.Buffered() > 0 {
			err = bw.Flush()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}
