package batchqueue

import "errors"

//https://github.com/smallnest/ringbuffer/blob/master/ring_buffer.go
var (
	ErrTooManyDataToWrite = errors.New("too many data to write")
	ErrIsFull             = errors.New("ringbuffer is full")
	ErrIsEmpty            = errors.New("ringbuffer is empty")
	ErrAccuqireLock       = errors.New("no lock to accquire")
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
	if r.w == r.r && !r.isFull {
		return 0
	}
	if r.w > r.r {
		return r.w - r.r
	}
	return r.size - r.r + r.w
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
		r.r = (r.r + n) % r.size
		return
	}

	n = r.size - r.r + r.w
	if n > len(p) {
		n = len(p)
	}

	if r.r+n <= r.size {
		copy(p, r.buf[r.r:r.r+n])
	} else {
		c1 := r.size - r.r
		copy(p, r.buf[r.r:r.size])
		c2 := n - c1
		copy(p[c1:], r.buf[0:c2])
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
