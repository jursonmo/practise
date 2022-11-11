package bufferpool

import "io"

type MyBuffer interface {
	io.ReadWriter
	ReadFrom(r io.Reader) (n int64, err error)
	Bytes() []byte
	Cap() int
}

type MyBufferPool interface {
	Get(int) MyBuffer
	Put(MyBuffer)
}

var defaultBufferPool MyBufferPool

func SetDefaultBufferPool(p MyBufferPool) {
	defaultBufferPool = p
}

func getMyBuf(n int) MyBuffer {
	return defaultBufferPool.Get(n)
	//return getBuffer(n)
}

func Release(b MyBuffer) {
	defaultBufferPool.Put(b)
}
