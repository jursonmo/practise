package unimsgcache

import (
	"errors"
	"sync"
)

var CacheFullErr = errors.New("cache is full")

//一个缓存：对于key 相同的数据，只保存一份, 比如上报一些数据时，对同一种数据，只需要上报最新的数据即可。
//Get 数据时随机取, 不分先后。 所以可以用GetAll读取所有msg, 如果msg处理成功，需要调用Ack 来删除缓存里的数据
//如果Ack时，缓存里的数据已经被更新过了(通过msg.seq 来判断)，就不能删除，因为它是新的数据。
type UniMsg struct {
	seq     uint32
	payload Msg //真正的message
}

func (um UniMsg) Msg() interface{} {
	return um.payload
}

type UniMsgCache struct {
	sync.Mutex
	seq   uint32
	max   int
	event chan struct{}
	cache map[string]UniMsg
}

// key 相同，表示同一种msg
type Msg interface {
	Key() string
}

func New() *UniMsgCache {
	return &UniMsgCache{}
}

func (q *UniMsgCache) Puts(msgs []Msg) error {
	//todo: check msg.Key() if empty
	q.Lock()
	for _, msg := range msgs {
		q.seq += 1
		q.cache[msg.Key()] = UniMsg{seq: q.seq, payload: msg}
	}
	q.Unlock()

	q.notify()
	return nil
}

func (q *UniMsgCache) Put(msg Msg) error {
	key := msg.Key()
	if key == "" {
		return errKeyEmpty
	}

	q.Lock()
	_, ok := q.cache[key]
	if !ok && q.max > 0 && len(q.cache) > q.max {
		q.Unlock()
		return CacheFullErr
	}
	q.seq += 1
	q.cache[key] = UniMsg{seq: q.seq, payload: msg}
	q.Unlock()

	q.notify()
	return nil
}

func (q *UniMsgCache) notify() {
	select {
	case q.event <- struct{}{}:
	default:
	}
}

func (q *UniMsgCache) REvent() chan struct{} {
	return q.event
}

var (
	errEmpty    = errors.New("empty")
	errKeyEmpty = errors.New("Key return empty")
)

func (q *UniMsgCache) Get() (UniMsg, error) {
	msgs := q.GetN(1)
	if msgs == nil {
		return UniMsg{}, errEmpty
	}
	return msgs[0], nil
}

// n == 0, means unlimit, get all msg from cache
func (q *UniMsgCache) GetN(n int) []UniMsg {
	q.Lock()
	defer q.Unlock()
	if len(q.cache) == 0 {
		return nil
	}

	if n == 0 || n > len(q.cache) {
		n = len(q.cache)
	}
	msgs := make([]UniMsg, 0, n)

	num := 0
	for _, msg := range q.cache {
		msgs = append(msgs, msg)
		num++
		if num >= n {
			break
		}
	}
	return msgs
}

//it can be blocked until there is msg return
func (q *UniMsgCache) BGetAll() []UniMsg {
	for {
		msgs := q.GetAll()
		if len(msgs) > 0 {
			return msgs
		}
		<-q.event
	}
}

func (q *UniMsgCache) GetAll() []UniMsg {
	return q.GetN(0)
}

//commit or Ack, means msg can't be remove from queue
func (q *UniMsgCache) Ack(msgs []UniMsg) {
	q.Lock()
	defer q.Unlock()
	for _, msg := range msgs {
		key := msg.payload.Key()
		uniMsg, ok := q.cache[key]
		if !ok {
			continue
		}
		if uniMsg.seq != msg.seq {
			continue
		}
		delete(q.cache, key)
	}
}
