package latestbykey

import (
	"context"
	"errors"
	"sync"

	"github.com/jursonmo/practise/pkg/backoffx"
	singletask "github.com/jursonmo/practise/pkg/singleTask"
)

//only want the latest data in each key:
//1. example kafka consumer commit msg offset, only commit highest offset every topic,partition,consumer group
//2. tcp ack
//3. i just need newest status

type Keyer interface {
	Key() string
}

type KeyTask struct {
	key  string
	task *singletask.SingleTask
}

type LatestByKey struct {
	ctx        context.Context
	once       sync.Once
	dataCh     chan []interface{}
	mu         sync.Mutex
	keytaskMap map[string]*KeyTask
	do         func(ctx context.Context, v interface{}) error
	err        error
	backoff    backoffx.Backoffer
	closeCh    chan struct{}
}

func NewLatestByKey(do func(ctx context.Context, v interface{}) error, backoff backoffx.Backoffer) *LatestByKey {
	l := &LatestByKey{do: do, backoff: backoff, dataCh: make(chan []interface{}, 128)}
	l.keytaskMap = make(map[string]*KeyTask)
	l.closeCh = make(chan struct{})

	return l
}

func (l *LatestByKey) Close() { close(l.closeCh) }

func (l *LatestByKey) Do(ctx context.Context, batch []interface{}) error {
	// l.once.Do(
	// 	func() {
	// 		l.start()
	// 	})
	l.dataCh <- batch
	return nil
}

func (l *LatestByKey) Start(ctx context.Context) error {
	l.ctx = ctx
	for {
		select {
		case <-l.closeCh:
			return errors.New("closed")
		case vv := <-l.dataCh:
			l.handleData(vv)
		}
	}
}

func (l *LatestByKey) handleData(vv []interface{}) {
	for _, v := range vv {
		k, ok := v.(Keyer)
		if !ok {
			continue
		}

		l.mu.Lock()

		key := k.Key()
		kt, ok := l.keytaskMap[key]
		if !ok {
			//add
			kt = &KeyTask{key: key, task: singletask.New(l.ctx)}
			l.keytaskMap[key] = kt
		}

		tmpv := v
		kt.task.PutTask(func(ctx context.Context) error {
			return l.do(ctx, tmpv)
		}, func(i interface{}) {
			if i == nil {
				//err is nil, means task executed successfully, so delete key from keytaskMap
				l.mu.Lock()
				defer l.mu.Unlock()
				delete(l.keytaskMap, key)
			}
		})

		l.mu.Unlock()
	}
}
