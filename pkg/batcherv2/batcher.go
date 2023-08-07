package batcher

// 带有去重，同步发送（通知消息发送结果）
import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-faster/city"
)

type CityHash struct{}

func (h *CityHash) Hash(s string) int {
	return int(city.Hash64([]byte(s)))
}

type Hasher interface {
	Hash(key string) int
}

var ErrFull = errors.New("channel is full")

type Option interface {
	apply(*options)
}

type options struct {
	size     int
	buffer   int
	worker   int
	interval time.Duration
	dedupe   bool //是否消息去重
}

//fixbug:
//func (o options) check() {
func (o *options) check() {
	if o.size <= 0 {
		o.size = 128
	}
	if o.buffer <= 0 {
		o.buffer = 128
	}
	if o.worker <= 0 {
		//o.worker = 5
		o.worker = 1 //默认只有一个
	}
	if o.interval <= 0 {
		o.interval = time.Second
	}
}

type funcOption struct {
	f func(*options)
}

func (fo *funcOption) apply(o *options) {
	fo.f(o)
}

func newOption(f func(*options)) *funcOption {
	return &funcOption{
		f: f,
	}
}

func WithSize(s int) Option {
	return newOption(func(o *options) {
		o.size = s
	})
}

func WithBuffer(b int) Option {
	return newOption(func(o *options) {
		o.buffer = b
	})
}

func WithWorker(w int) Option {
	return newOption(func(o *options) {
		o.worker = w
	})
}

func WithInterval(i time.Duration) Option {
	return newOption(func(o *options) {
		o.interval = i
	})
}

func WithDedupe(b bool) Option {
	return newOption(func(o *options) {
		o.dedupe = b
	})
}

type Msg struct {
	key string
	val interface{}
	//for syn
	done chan struct{} //用于同步阻塞模式下，通知消息处理结果
	err  error
}

var (
	ErrOverWrited = errors.New("Overwrited")
)

func NewMsg(key string, val interface{}) *Msg {
	//todo:
	return &Msg{key: key, val: val, done: make(chan struct{}, 1)}
}

func NewMsgAsyn(key string, val interface{}) *Msg {
	//todo:
	return &Msg{key: key, val: val}
}

func (m *Msg) Key() string {
	return m.key
}

func (m *Msg) Value() interface{} {
	return m.val
}

//wait for msg complete
func (m *Msg) Wait() error {
	defer m.Release()
	if m.done != nil {
		<-m.done
	}
	return m.err
}

func (m *Msg) Complete(err error) {
	if m.done != nil {
		m.err = err
		close(m.done)
	}
}
func (m *Msg) Release() {
	//todo: reset and put back sync.Pool
}

type Exector interface {
	Do(ctx context.Context, val map[string][]interface{}) error
}

type Batcher struct {
	opts options

	exector  Exector
	Sharding Hasher
	chans    []chan *Msg
	wait     sync.WaitGroup
}

func New(e Exector, opts ...Option) *Batcher {
	b := &Batcher{exector: e}
	for _, opt := range opts {
		opt.apply(&b.opts)
	}
	b.opts.check()

	b.chans = make([]chan *Msg, b.opts.worker)
	for i := 0; i < b.opts.worker; i++ {
		b.chans[i] = make(chan *Msg, b.opts.buffer)
	}
	return b
}
func (b *Batcher) String() string {
	o := b.opts
	return fmt.Sprintf("size:%d, buffer:%d, worker:%d, intvl:%v, dedupe:%v", o.size, o.buffer, o.worker, o.interval, o.dedupe)
}

func (b *Batcher) Start() {
	if b.exector == nil {
		log.Fatal("Batcher: Do func is nil")
	}
	if b.Sharding == nil {
		b.Sharding = &CityHash{}
	}
	b.wait.Add(len(b.chans))
	for i, ch := range b.chans {
		go b.merge(i, ch)
	}
}

func (b *Batcher) Addx(key string, val interface{}, sync bool) error {
	ch, msg := b.add(key, val, sync)
	select {
	case ch <- msg:
	default:
		return ErrFull
	}
	return msg.Wait()
}

//同步, 阻塞,等待处理结果
func (b *Batcher) Add(key string, val interface{}) error {
	return b.Addx(key, val, true)
}

//异步，直接返回
func (b *Batcher) AddAsyn(key string, val interface{}) error {
	return b.Addx(key, val, false)
}

func (b *Batcher) add(key string, val interface{}, sync bool) (chan *Msg, *Msg) {
	sharding := b.Sharding.Hash(key) % b.opts.worker
	ch := b.chans[sharding]
	var msg *Msg
	if sync {
		msg = NewMsg(key, val)
	} else {
		msg = NewMsgAsyn(key, val)
	}
	return ch, msg
}

func (b *Batcher) merge(idx int, ch <-chan *Msg) {
	defer b.wait.Done()

	var (
		m          *Msg
		count      int
		closed     bool
		lastTicker = true
		interval   = b.opts.interval
		msgs       = make(map[string][]*Msg, b.opts.size)
	)
	if idx > 0 {
		interval = time.Duration(int64(idx) * (int64(b.opts.interval) / int64(b.opts.worker)))
	}
	ticker := time.NewTicker(interval)
	for {
		select {
		case m = <-ch:
			if m == nil {
				closed = true
				break
			}
			count++
			if b.opts.dedupe {
				//去重，把之前的消息剔除
				if oldMsgs, ok := msgs[m.key]; ok {
					oldMsgs[0].Complete(ErrOverWrited)
					delete(msgs, m.key)
				}
			}
			msgs[m.key] = append(msgs[m.key], m)

			if count >= b.opts.size {
				break
			}
			continue
		case <-ticker.C:
			if lastTicker {
				ticker.Stop()
				ticker = time.NewTicker(b.opts.interval)
				lastTicker = false
			}
		}
		if len(msgs) > 0 {
			ctx := context.Background()

			//把msg 转成 map[string][]interface{}
			data := make(map[string][]interface{})
			for key, msgx := range msgs {
				for _, msg := range msgx {
					data[key] = append(data[key], msg.Value())
				}
			}
			err := b.exector.Do(ctx, data) // 不用管处理是否失败吗？
			//反馈处理结果
			for _, msgx := range msgs {
				for _, msg := range msgx {
					msg.Complete(err)
				}
			}
			msgs = make(map[string][]*Msg, b.opts.size)
			count = 0
		}
		if closed {
			ticker.Stop()
			return
		}
	}
}

func (b *Batcher) Close() {
	for _, ch := range b.chans {
		ch <- nil // 通过发送nil 来终止任务， 而不是close(ch), 避免向ch 写数据panic
	}
	b.wait.Wait()
}
