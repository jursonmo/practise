package batchtask

import (
	"context"
	"sync"

	"github.com/jursonmo/practise/pkg/batchqueue"
)

type Option func(*Task)

func WithBufferRoll(b bool) Option {
	return func(t *Task) {
		t.roll = b
	}
}
func WithMaxBatchSize(size int) Option {
	return func(t *Task) {
		t.maxBatchSize = size
	}
}

type Task struct {
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	name     string
	buffer   *batchqueue.BatchQueue
	capacity int
	roll     bool //buffer roll

	executor     Executor
	maxBatchSize int //max size of do()

	closed bool
	mu     sync.Mutex
}

type Executor interface {
	Do(ctx context.Context, batch []interface{}) error
	Close()
}

func NewTask(name string, capacity int, e Executor, opts ...Option) *Task {
	task := &Task{
		name:     name,
		capacity: capacity,
		executor: e,
		buffer:   batchqueue.NewBatchQueue(capacity, batchqueue.WithName(name+"_queue")),
	}
	for _, opt := range opts {
		opt(task)
	}
	if task.maxBatchSize == 0 {
		task.maxBatchSize = 1
	}

	return task
}

func (t *Task) Add(data ...interface{}) (n int, err error) {
	if t.roll {
		n, err = t.buffer.PutRoll(data...)
	} else {
		n, err = t.buffer.Put(data...)
	}
	if err != nil {
		return
	}
	return
}

func (t *Task) getLoop(ctx context.Context) {
	defer t.Stop()
	for {
		entires, err := t.buffer.GetWithSize(t.maxBatchSize)
		if err != nil {
			return
		}
		t.executor.Do(ctx, entires)
	}
}

func (t *Task) Start(ctx context.Context) error {
	t.ctx, t.cancel = context.WithCancel(ctx)
	go t.getLoop(ctx)
	go func() error {
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}()

	return nil
}

func (t *Task) Stop() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
	}
	if t.buffer != nil {
		t.buffer.Close()
	}
	if t.executor != nil {
		t.executor.Close()
	}
}
