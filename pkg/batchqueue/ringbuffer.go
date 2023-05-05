package batchqueue

import (
	"errors"
	"log"
	"sync"
)

//https://github.com/smallnest/ringbuffer/blob/master/ring_buffer.go
var (
	ErrTooManyDataToWrite = errors.New("too many data to write")
	ErrIsFull             = errors.New("ringbuffer is full")
	ErrIsEmpty            = errors.New("ringbuffer is empty")
	ErrAccuqireLock       = errors.New("no lock to accquire")
	ErrOverCapacity       = errors.New("Over Capacity")
)

type RingBuffer struct {
	buf    []interface{}
	size   int
	r      int // next position to read
	w      int // next position to write
	isFull bool
	mu     Mutex
}

type Mutex interface {
	Lock()
	Unlock()
}

type NonMutex struct{}

func (n *NonMutex) Lock()   {}
func (n *NonMutex) Unlock() {}

// New returns a new RingBuffer whose buffer has the given size.
func New(size int, opts ...Option) *RingBuffer {
	rb := &RingBuffer{
		buf:  make([]interface{}, size),
		size: size,
	}
	o := Options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.withoutMutex {
		rb.mu = &NonMutex{}
	}

	//if mu unset, default sync.Mutex
	if rb.mu == nil {
		rb.mu = new(sync.Mutex)
	}
	return rb
}

type Options struct {
	withoutMutex bool
}
type Option func(*Options)

func WithoutMutex(b bool) Option {
	return func(o *Options) {
		o.withoutMutex = b
	}
}

func (r *RingBuffer) Read(p []interface{}) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	n, err = r.read(p)
	r.mu.Unlock()

	return n, err
}

func (r *RingBuffer) Buffered() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buffered()
}

func (r *RingBuffer) buffered() int {
	if r.w == r.r && !r.isFull {
		return 0
	}
	if r.w > r.r {
		return r.w - r.r
	}
	return r.size - r.r + r.w
}

func (r *RingBuffer) free() int {
	return r.size - r.buffered()
}

func (r *RingBuffer) Discard(dn int) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.discard(dn)
}

func (r *RingBuffer) delElement(from, to int) {
	for i := from; i < to; i++ {
		r.buf[i] = nil //make it gc
	}
}

func (r *RingBuffer) discard(dn int) (n int, err error) {
	if dn == 0 {
		return
	}
	if r.w == r.r && !r.isFull {
		return 0, ErrIsEmpty
	}
	if r.w > r.r {
		n = r.w - r.r
		if n > dn {
			n = dn
		}
		r.delElement(r.r, r.r+n)
		r.r = (r.r + n) % r.size
		return
	}

	n = r.size - r.r + r.w
	if n > dn {
		n = dn
	}
	//----------
	c1 := r.size - r.r
	if c1 >= dn {
		r.delElement(r.r, r.r+dn)
	} else {
		r.delElement(r.r, r.size)
		c2 := 0
		if dn-c1 < r.w {
			c2 = dn - c1
		} else {
			c2 = r.w
		}
		r.delElement(0, c2)
	}
	//-------------
	r.r = (r.r + n) % r.size
	r.isFull = false
	return
}

func (r *RingBuffer) read(p []interface{}) (n int, err error) {
	if r.w == r.r && !r.isFull {
		return 0, ErrIsEmpty
	}

	if r.w > r.r {
		n = r.w - r.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, r.buf[r.r:r.r+n])
		r.delElement(r.r, r.r+n)
		r.r = (r.r + n) % r.size
		return
	}

	n = r.size - r.r + r.w
	if n > len(p) {
		n = len(p)
	}

	if r.r+n <= r.size {
		copy(p, r.buf[r.r:r.r+n])
		r.delElement(r.r, r.r+n)
	} else {
		c1 := r.size - r.r
		copy(p, r.buf[r.r:r.size])
		r.delElement(r.r, r.size)
		c2 := n - c1
		copy(p[c1:], r.buf[0:c2])
		r.delElement(0, c2)
	}
	r.r = (r.r + n) % r.size

	r.isFull = false

	return n, err
}

func (r *RingBuffer) Write(p []interface{}) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	n, err = r.write(p)
	r.mu.Unlock()

	return n, err
}

//1. if writing data bigger than the capacity of ringbuffer, return err
//2. it will replace oldest data when there is no enough space to write
func (r *RingBuffer) WriteRoll(p []interface{}) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(p) > r.size {
		return 0, ErrOverCapacity
	}

	free := r.free()
	if free < len(p) {
		needDiscard := len(p) - free
		r.discard(needDiscard)
	}

	n, err = r.write(p)
	if n != len(p) {
		log.Fatalf("WriteCover fail, len(p):%d, n:%d", len(p), n)
	}

	return n, err
}

func (r *RingBuffer) write(p []interface{}) (n int, err error) {
	if r.isFull {
		return 0, ErrIsFull
	}

	var avail int
	if r.w >= r.r {
		avail = r.size - r.w + r.r
	} else {
		avail = r.r - r.w
	}

	if len(p) > avail {
		err = ErrTooManyDataToWrite
		p = p[:avail]
	}
	n = len(p)

	if r.w >= r.r {
		c1 := r.size - r.w
		if c1 >= n {
			copy(r.buf[r.w:], p)
			r.w += n
		} else {
			copy(r.buf[r.w:], p[:c1])
			c2 := n - c1
			copy(r.buf[0:], p[c1:])
			r.w = c2
		}
	} else {
		copy(r.buf[r.w:], p)
		r.w += n
	}

	if r.w == r.size {
		r.w = 0
	}
	if r.w == r.r {
		r.isFull = true
	}

	return n, err
}
