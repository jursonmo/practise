package fixedpool

import (
	"errors"
	"io"
	"sync"

	bufferpool "github.com/jursonmo/practise/pkg/bufferPool"
)

type MyBuffer = bufferpool.MyBuffer

const fixedBufferSize int = 1600

type Buffer struct {
	offset int
	buf    [fixedBufferSize]byte
}

type fpool struct {
	sync.Pool
}

func InitFixedPool() {
	pool := &fpool{Pool: sync.Pool{New: func() interface{} { return &Buffer{} }}}
	bufferpool.SetDefaultBufferPool(pool)
}

func (fp *fpool) Get(n int) MyBuffer {
	b := fp.Pool.Get().(*Buffer)
	b.Reset()
	return b
}

func (fp *fpool) Put(b MyBuffer) {
	fp.Pool.Put(b)
}

var ErrNotEnough = errors.New("ErrNotEnough")

func (b *Buffer) Write(buf []byte) (int, error) {
	if len(buf) > b.free() {
		return 0, ErrNotEnough
	}
	n := copy(b.buf[b.offset:], buf)
	b.offset += n
	return n, nil
}

func (b *Buffer) Read(buf []byte) (int, error) {
	n := copy(buf, b.buf[:b.offset])
	return n, nil
}

func (b *Buffer) free() int {
	return cap(b.buf) - b.offset
}

func (b *Buffer) Bytes() []byte {
	return b.buf[:b.offset]
}

func (b *Buffer) Cap() int {
	return cap(b.buf)
}

func (b *Buffer) Reset() {
	b.offset = 0
}

func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	rn, err := r.Read(b.buf[:])
	if err != nil {
		return int64(rn), err
	}
	b.offset += rn
	return int64(rn), err
}
