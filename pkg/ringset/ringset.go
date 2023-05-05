package ringset

//a ringBuffer store unique elements, like a set
//To distinguish different elements by their Key()
// use the Key() of an element to differentiate it from others
// so element must implement Keyer

import (
	"errors"
	"fmt"
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
	ErrElement            = errors.New("element invalid")
)

type Keyer interface {
	Key() string
}

type RingBuffer struct {
	buf    []interface{}
	smap   map[string]int
	isSet  bool
	size   int //cap
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
func New(cap int, opts ...Option) *RingBuffer {
	rb := &RingBuffer{
		buf:  make([]interface{}, cap),
		size: cap,
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

	rb.isSet = o.withIsset
	if rb.isSet {
		rb.smap = make(map[string]int)
	}
	return rb
}

func (r *RingBuffer) String() string {
	return fmt.Sprintf("cap:%d, r:%d, w:%d, isSet:%v, len(smap):%d", r.size, r.r, r.w, r.isSet, len(r.smap))
}

type Options struct {
	withoutMutex bool
	withIsset    bool
}

type Option func(*Options)

func WithoutMutex(b bool) Option {
	return func(o *Options) {
		o.withoutMutex = b
	}
}

func WithIsSet(b bool) Option {
	return func(o *Options) {
		o.withIsset = b
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

func (r *RingBuffer) setMap(from, to int) {
	for i := from; i < to; i++ {
		if r.isSet {
			e := r.buf[i]
			r.smap[e.(Keyer).Key()] = i
		}
	}
}

func (r *RingBuffer) delElement(from, to int) {
	for i := from; i < to; i++ {
		if r.isSet {
			e := r.buf[i]
			delete(r.smap, e.(Keyer).Key())
		}
		r.buf[i] = nil //make it gc
	}
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

	if r.isSet {
		for _, pe := range p {
			if _, ok := pe.(Keyer); !ok {
				return 0, ErrElement
			}
		}
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

	if r.isSet {
		for _, pe := range p {
			if _, ok := pe.(Keyer); !ok {
				return 0, ErrElement
			}
		}
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

//注意：先按顺序判断是否已经在set 里，如果发现某个元素不在集合里，这时发现没有空闲的位置存放这个新元素，就不在往后判断,即退出
//因为返回值是有short write 的情况的，即返回值 n > 0, err != nil, 如果不按顺序来，
//应用层不知道哪些元素已经写入集合里，哪些没有，所以必须按顺序来，
func (r *RingBuffer) write(p []interface{}) (n int, err error) {
	// if r.isFull {
	// 	return 0, ErrIsFull
	// }

	var avail int
	if r.isFull {
		avail = 0
	} else if r.w >= r.r {
		avail = r.size - r.w + r.r
	} else {
		avail = r.r - r.w
	}

	repeat := 0
	if r.isSet {
		tmp := make([]interface{}, 0, len(p))
		for _, pi := range p {
			pe := pi.(Keyer)
			key := pe.Key()
			if position, ok := r.smap[key]; ok {
				r.buf[position] = pi //update
				repeat++
			} else {
				if len(tmp) == avail {
					return repeat, ErrTooManyDataToWrite
				}
				tmp = append(tmp, pi)
			}
		}
		p = tmp
		if len(p) == 0 {
			return repeat, nil
		}
	}

	//走到这里，说明有新的不同的元素加进来，这时就需要判断是否full 了
	if r.isFull {
		return 0, ErrIsFull
	}

	//如果是isSet is true, 下面这个判断是不会成立的
	if len(p) > avail {
		// err = ErrTooManyDataToWrite
		// p = p[:avail]
		//对于新的元素，不支持short write ,直接返回，因为如果支持short write, 其实应用层不知道哪些元素write 进来，哪些没有
		//log.Printf("len(p):%d > avail:%d, r.size:%d, r.w:%d, r.r:%d \n", len(p), avail, r.size, r.w, r.r)
		return repeat, ErrTooManyDataToWrite
	}
	n = len(p)

	if r.w >= r.r {
		c1 := r.size - r.w
		if c1 >= n {
			copy(r.buf[r.w:], p)
			r.setMap(r.w, r.w+n)
			r.w += n
		} else {
			copy(r.buf[r.w:], p[:c1])
			r.setMap(r.w, r.w+c1)
			c2 := n - c1
			copy(r.buf[0:], p[c1:])
			r.setMap(0, c2)
			r.w = c2
		}
	} else {
		copy(r.buf[r.w:], p)
		r.setMap(r.w, r.w+n)
		r.w += n
	}

	if r.w == r.size {
		r.w = 0
	}
	if r.w == r.r {
		r.isFull = true
	}

	return n + repeat, err
}
