package batchqueue

import (
	"errors"
	"fmt"
	"sync"
)

//github.com/segmentio/kafka-go@v0.4.32/writer.go
type batchQueue struct {
	name string
	rb   *RingBuffer
	//queue []interface{}

	// Pointers are used here to make `go vet` happy, and avoid copying mutexes.
	// It may be better to revert these to non-pointers and avoid the copies in
	// a different way.
	mutex *sync.Mutex
	cond  *sync.Cond

	closed   bool
	closeErr error
}

var (
	errClose = errors.New("closed")
)

func NewBatchQueue(initialSize int, name string) batchQueue {
	bq := batchQueue{
		name: name,
		//queue: make([]interface{}, 0, initialSize),
		rb:    New(initialSize, WithoutMutex(true)),
		mutex: &sync.Mutex{},
		cond:  &sync.Cond{},
	}

	bq.cond.L = bq.mutex

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
	n, _ := b.rb.Write(batch)
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
	if b.rb.Buffered() == 0 {
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
	for b.rb.Buffered() == 0 && !b.closed {
		b.cond.Wait()
	}

	if b.rb.Buffered() == 0 && b.closed {
		return nil, b.closeErr
	}

	bufferNum := b.rb.Buffered()
	if bufferNum == 0 {
		panic("bufferNum == 0")
	}
	if n > bufferNum {
		n = bufferNum
	}
	need := make([]interface{}, n)
	rn, err := b.rb.Read(need)
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
	b.closeErr = fmt.Errorf("%s:%w", b.name, errClose)
}
