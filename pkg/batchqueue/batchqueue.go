package batchqueue

import (
	"errors"
	"fmt"
	"sync"
)

type BatchQueue = batchQueue
type BqOptions struct {
	name string
	roll bool
}
type BqOption func(*BqOptions)

func WithName(name string) BqOption {
	return func(o *BqOptions) {
		o.name = name
	}
}
func WithRoll(roll bool) BqOption {
	return func(o *BqOptions) {
		o.roll = roll
	}
}

//借鉴github.com/segmentio/kafka-go@v0.4.32/writer.go 但是它会分配新的内存块
type batchQueue struct {
	//b *RingBuffer
	buf Buffer
	//queue []interface{}
	option BqOptions
	// Pointers are used here to make `go vet` happy, and avoid copying mutexes.
	// It may be better to revert these to non-pointers and avoid the copies in
	// a different way.
	mutex *sync.Mutex
	cond  *sync.Cond

	closed   bool
	closeErr error
}

type Buffer interface {
	Read(p []interface{}) (n int, err error)
	Write(p []interface{}) (n int, err error)
	WriteRoll(p []interface{}) (n int, err error)
	Buffered() int
	Discard(dn int) (n int, err error)
}

var (
	errClose = errors.New("closed")
)

func NewBatchQueue(initialSize int, opts ...BqOption) *batchQueue {
	bq := &batchQueue{
		//queue: make([]interface{}, 0, initialSize),
		buf:   New(initialSize, WithoutMutex(true)),
		mutex: &sync.Mutex{},
		cond:  &sync.Cond{},
	}
	bq.cond.L = bq.mutex

	for _, opt := range opts {
		opt(&bq.option)
	}
	return bq
}

//only batchQueue closed can return a non-nil err
func (b *batchQueue) Put(batch ...interface{}) (int, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	defer b.cond.Broadcast()

	if b.closed {
		return 0, b.closeErr
	}
	//b.queue = append(b.queue, batch)
	n, _ := b.buf.Write(batch)
	return n, nil
}

//it will replace oldest data when there is no enough space to write
func (b *batchQueue) PutRoll(batch ...interface{}) (int, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	defer b.cond.Broadcast()

	if b.closed {
		return 0, b.closeErr
	}

	n, err := b.buf.WriteRoll(batch)
	if err == ErrOverCapacity {
		return n, err
	}
	return n, nil
}

//only batchQueue closed can return a non-nil err
func (b *batchQueue) Get() (interface{}, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	vv, err := b.getWithSize(1)
	if errors.Is(err, errClose) {
		return nil, err
	}
	return vv[0], nil
}

func (b *batchQueue) TryGet() (interface{}, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	if b.closed {
		return nil, b.closeErr
	}
	if b.buf.Buffered() == 0 {
		return nil, nil
	}

	vv, err := b.getWithSize(1)
	if errors.Is(err, errClose) {
		return nil, err
	}
	return vv[0], nil
}

//return a non-nil err when batchQueue closed
func (b *batchQueue) GetWithSize(n int) ([]interface{}, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	vv, err := b.getWithSize(n)
	if errors.Is(err, errClose) {
		return nil, err
	}
	return vv, nil
}

//drain the ringbuffer and then check if batchQueue is closed
func (b *batchQueue) getWithSize(n int) ([]interface{}, error) {
	for b.buf.Buffered() == 0 && !b.closed {
		b.cond.Wait()
	}

	if b.buf.Buffered() == 0 && b.closed {
		return nil, b.closeErr
	}

	bufferNum := b.buf.Buffered()
	if bufferNum == 0 {
		panic("bufferNum == 0")
	}
	if n > bufferNum {
		n = bufferNum
	}
	need := make([]interface{}, n)
	rn, err := b.buf.Read(need)
	if len(need) != rn {
		panic("len(need) != rn")
	}
	return need, err
}

func (b *batchQueue) Close() {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	defer b.cond.Broadcast()

	b.closed = true
	b.closeErr = fmt.Errorf("%s:%w", b.option.name, errClose)
}
