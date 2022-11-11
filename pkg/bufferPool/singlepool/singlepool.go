package singlepool

import (
	"bytes"
	"sync"

	bufferpool "github.com/jursonmo/practise/pkg/bufferPool"
)

type spool struct {
	sync.Pool
}

var pool *spool

func InitSinglePool(defaultBufSize int) {
	pool = &spool{Pool: sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, defaultBufSize)) }}}
	bufferpool.SetDefaultBufferPool(pool)
}

func (sp *spool) Get(n int) bufferpool.MyBuffer {
	b := sp.Pool.Get().(*bytes.Buffer)
	b.Reset()

	return b
}

func (sp *spool) Put(b bufferpool.MyBuffer) {
	sp.Pool.Put(b)
}
