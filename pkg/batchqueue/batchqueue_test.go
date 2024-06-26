package batchqueue

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestPutGet(t *testing.T) {
	size := 10
	entry := int(1)
	batchqueue := NewBatchQueue(size, WithName("TestPutGet_queue"))
	n, err := batchqueue.Put(entry)
	if err != nil || n != 1 {
		t.Fatal(err)
	}

	v, err := batchqueue.Get()
	if err != nil || v.(int) != entry {
		t.Fatal(err)
	}

	batchqueue.Close()
	_, err = batchqueue.Put(entry)
	if err == nil {
		t.Fatalf("batchqueue is closed, shouldn't put entry")
	}
}

func TestTryGet(t *testing.T) {
	size := 10
	entry := int(1)
	batchqueue := NewBatchQueue(size, WithName("TestTryGet_queue"))

	//TryGet: unblock get
	v, err := batchqueue.TryGet()
	if err != nil || v != nil {
		t.Fatal(err)
	}

	n, err := batchqueue.Put(entry)
	if err != nil || n != 1 {
		t.Fatal(err)
	}

	v, err = batchqueue.TryGet()
	if err != nil || v.(int) != entry {
		t.Fatal(err)
	}

	batchqueue.Close()
}

func TestPut(t *testing.T) {
	size := 10
	entry := int(1)
	batchqueue := NewBatchQueue(size, WithName("TestPut_queue"))

	for i := 0; i < size; i++ {
		n, err := batchqueue.Put(i)
		if err != nil || n != 1 {
			t.Fatal(err)
		}
	}

	n, err := batchqueue.Put(entry)
	if n > 0 {
		t.Fatal("unexpect: batchqueue is full, but put any entry in batchqueue again")
	}
	if err != nil {
		t.Fatal(err)
	}

	batchqueue.Close()
	_, err = batchqueue.Put(entry)
	if err == nil {
		t.Fatalf("batchqueue is closed, shouldn't put entry")
	}
}

func TestMustPut(t *testing.T) {
	size := 10
	batchqueue := NewBatchQueue(size, WithName("TestMustPut_queue"))

	for i := 0; i < size; i++ {
		n, err := batchqueue.Put(i)
		if err != nil || n != 1 {
			t.Fatal(err)
		}
	}

	//data 11 repalce the data:0
	entry := int(11)
	n, err := batchqueue.PutRoll(entry)
	if n == 0 || err != nil {
		t.Fatal("unexpect: MustPut should write data to a full queue")
	}

	//now data is 1....11, so the next data is 1
	v, err := batchqueue.Get()
	if err != nil || v.(int) != 1 {
		t.Fatal(err)
	}

	//get two data 2..3
	_, err = batchqueue.GetWithSize(2)
	if err != nil {
		t.Fatal(err)
	}
	//write four data: 12...15, but there is three place to write, so it will replace data 4
	entries := make([]interface{}, 0, 4)
	for i := 12; i < 16; i++ {
		entries = append(entries, i)
	}
	n, err = batchqueue.PutRoll(entries...)
	if n != len(entries) {
		t.Fatalf("n:%d ,len(entries):%d\n", n, len(entries))
	}
	//now the next data should be 5
	v, err = batchqueue.Get()
	if err != nil || v.(int) != 5 {
		t.Fatal(err)
	}

	batchqueue.Close()
}

//put one by one,  get one by one concurrently
func TestGet(t *testing.T) {
	size := 1000
	batch := NewBatchQueue(size, WithName("TestGet_queue"))

	wg := sync.WaitGroup{}
	wg.Add(1)
	getSum := 0
	go func() {
		defer wg.Done()

		for {
			v, err := batch.Get()
			if err != nil {
				t.Log(err)
				return
			}
			//t.Logf("get, v:%v", v)
			getSum += v.(int)
		}
	}()

	putSum := 0
	for i := 0; i < size; i++ {
		n, err := batch.Put(i)
		if err != nil {
			t.Fatal(err)
		}
		_ = n
		//t.Logf("put, v:%d, n:%d\n", i, n)
		putSum += i
	}
	batch.Close()

	wg.Wait()
	if getSum == 0 || getSum != putSum {
		t.Fatalf("get sum :=%d", getSum)
	}
}

//put one by one, batch get concurrently
func TestGetWithSize(t *testing.T) {
	size := 1000
	batch := NewBatchQueue(size, WithName("TestGetWithSize_queue"))

	wg := sync.WaitGroup{}
	wg.Add(1)
	getSum := 0
	go func() {
		defer wg.Done()

		for {
			vv, err := batch.GetWithSize(10)
			if err != nil {
				t.Log(err)
				return
			}
			//t.Logf("get, v:%v", v)
			for _, v := range vv {
				getSum += v.(int)
			}
		}
	}()

	putSum := 0
	for i := 0; i < size; i++ {
		n, err := batch.Put(i)
		if err != nil {
			t.Fatal(err)
		}
		_ = n
		//t.Logf("put, v:%d, n:%d\n", i, n)
		putSum += i
	}
	batch.Close()

	wg.Wait()
	if getSum == 0 || getSum != putSum {
		t.Fatalf("get sum :=%d", getSum)
	}
}

//batch put , batch get concurrently
func TestPutBatch(t *testing.T) {
	batchSize := 20
	size := batchSize * 100
	batchqueue := NewBatchQueue(size, WithName("TestPutBatch_queue"))

	wg := sync.WaitGroup{}
	wg.Add(1)
	getSum := 0
	go func() {
		defer wg.Done()

		for {
			vv, err := batchqueue.GetWithSize(10)
			if err != nil {
				t.Log(err)
				return
			}
			//t.Logf("get, vv:%v", vv)
			for _, v := range vv {
				getSum += v.(int)
			}
		}
	}()

	putSum := 0
	batchEntry := make([]interface{}, 0, batchSize)

	for i := 0; i < size; i++ {
		if len(batchEntry) < batchSize {
			batchEntry = append(batchEntry, i)
		}
		if len(batchEntry) < batchSize {
			continue
		}

		//now: len(batchEntry) == batchSize
		n, err := batchqueue.Put(batchEntry...)
		if err != nil {
			t.Fatal(err)
		}

		//t.Logf("put,i:%d,n:%d\n", i, n)
		for _, entry := range batchEntry[:n] {
			putSum += entry.(int)
		}

		//reset
		batchEntry = batchEntry[0:0:batchSize]
	}
	t.Logf("putSum:%d", putSum)
	batchqueue.Close()

	wg.Wait()
	if getSum == 0 || getSum != putSum {
		t.Fatalf("get sum :=%d", getSum)
	}
}

//multi puter, and multi geter
func TestMultiPutGet(t *testing.T) {
	size := 100
	batch := NewBatchQueue(size, WithName("TestMultiPutGet_queue"))

	wg_get := sync.WaitGroup{}
	wg_put := sync.WaitGroup{}

	getSum := int64(0)
	geter := func() {
		defer wg_get.Done()

		for {
			v, err := batch.Get()
			if err != nil {
				t.Log(err)
				return
			}
			//t.Logf("get, v:%v", v)
			atomic.AddInt64(&getSum, int64(v.(int)))
		}
	}

	geterNum := 10
	wg_get.Add(geterNum)
	for i := 0; i < geterNum; i++ {
		go geter()
	}

	putSum := int64(0)
	puterNum := 10
	wg_put.Add(puterNum)

	puter := func(start, end int) {
		defer wg_put.Done()
		//t.Logf("start:%d, end:%d", start, end)
		for i := start; i < end; i++ {
			n, err := batch.Put(i)
			if err != nil {
				t.Fatal(err)
			}
			_ = n
			//t.Logf("put, v:%d, n:%d\n", i, n)
			atomic.AddInt64(&putSum, int64(i))
		}
	}

	quantum := size / puterNum
	for i := 0; i < puterNum; i++ {
		go puter(i*quantum, (i+1)*quantum)
	}
	wg_put.Wait()
	batch.Close()

	wg_get.Wait()
	t.Logf("get sum :=%d", atomic.LoadInt64(&getSum))
	if atomic.LoadInt64(&getSum) == 0 || atomic.LoadInt64(&getSum) != atomic.LoadInt64(&putSum) {
		t.Fatalf("get sum :=%d", atomic.LoadInt64(&getSum))
	}
}
