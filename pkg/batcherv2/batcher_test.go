package batcher

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type dedupeWorker struct{}

var _ Exector = (*dedupeWorker)(nil)

func (w *dedupeWorker) Do(ctx context.Context, valsMap map[string][]interface{}) error {
	for key, vals := range valsMap {
		if len(vals) > 1 {
			panic("")
		}
		fmt.Printf("key:%v, value:%v\n", key, vals[0])
	}
	return nil
}

func TestPutGet(t *testing.T) {
	worker := &dedupeWorker{}
	batcher := New(worker, WithDedupe(true), WithInterval(time.Second))
	batcher.Start(context.Background())
	t.Logf("batcher:%v", batcher)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	var err1 error
	go func() {
		defer wg.Done()
		err1 = batcher.Add("key", "value1")
	}()
	time.Sleep(time.Millisecond * 20)

	err2 := batcher.Add("key", "value2")
	if err2 != nil {
		t.Fatal("expect value2 ok")
	}
	wg.Wait()

	if err1 == nil {
		t.Fatal("expect add value1 return err of overwrited ")
	}
	t.Logf("err1:%v", err1)
}
