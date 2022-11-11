package pool

import (
	"bytes"
	"errors"
	"os"
	"strconv"
	"sync"

	bufferpool "github.com/jursonmo/practise/pkg/bufferPool"
)

type MyBuffer = bufferpool.MyBuffer

var (
	errWrongSize = errors.New("wrong size")
)

var (
	envMinSize = os.Getenv("pool_min_size")
	envMaxSize = os.Getenv("pool_max_size")
	envFactor  = os.Getenv("pool_factor")

	defaultMinSize = 512
	defaultMaxSize = 2 * 1024
	defaultFactor  = 2

	DefaultPool *SyncPool
)

func init() {
	if minSize, err := strconv.Atoi(envMinSize); err == nil {
		defaultMinSize = minSize
	}
	if maxSize, err := strconv.Atoi(envMaxSize); err == nil {
		defaultMaxSize = maxSize
	}
	if factor, err := strconv.Atoi(envFactor); err == nil {
		defaultFactor = factor
	}

	DefaultPool, _ = NewSyncPool(defaultMinSize, defaultMaxSize, defaultFactor)
}

func InitMultiPool(minSize, maxSize, factor int) error {
	pool, err := NewSyncPool(minSize, maxSize, factor)
	if err != nil {
		return err
	}
	bufferpool.SetDefaultBufferPool(pool)
	return nil
}

// SyncPool is a sync.Pool base slab allocation memory pool
type SyncPool struct {
	classes     []sync.Pool
	classesSize []int
	minSize     int
	maxSize     int
}

// NewSyncPool create a sync.Pool base slab allocation memory pool.
// minSize is the smallest chunk size.
// maxSize is the lagest chunk size.
// factor is used to control growth of chunk size.
func NewSyncPool(minSize, maxSize, factor int) (*SyncPool, error) {
	if minSize <= 0 || maxSize <= 0 || factor <= 1 || maxSize < minSize {
		return nil, errWrongSize
	}
	n := 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		n++
	}
	pool := &SyncPool{
		make([]sync.Pool, n),
		make([]int, n),
		minSize, maxSize,
	}
	n = 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		pool.classesSize[n] = chunkSize
		pool.classes[n].New = func(size int) func() interface{} {
			return func() interface{} {
				buf := make([]byte, size)
				// return &buf
				return bytes.NewBuffer(buf)
			}
		}(chunkSize)
		n++
	}
	return pool, nil
}

// Alloc try alloc a []byte from internal slab class if no free chunk in slab class Alloc will make one.
func (pool *SyncPool) Get(size int) MyBuffer {
	if size < 0 {
		size = pool.maxSize
	}
	if size <= pool.maxSize {
		for i := 0; i < len(pool.classesSize); i++ {
			if pool.classesSize[i] >= size {
				b := pool.classes[i].Get().(*bytes.Buffer)
				b.Reset()
				return b
			}
		}
	}
	return bytes.NewBuffer(make([]byte, size))
}

// Free release a []byte that alloc from Pool.Alloc.
func (pool *SyncPool) Put(b MyBuffer) {
	if b == nil {
		return
	}
	if size := b.Cap(); size >= pool.minSize && size <= pool.maxSize {
		// for i := range pool.classesSize {
		// 	if pool.classesSize[i] >= size {
		// 		pool.classes[i].Put(b)
		// 		return
		// 	}
		// }
		//放回去的时候，要看size, 为了避免业务使用时发生扩容，放在size 的下一级
		//比如 pool 是三个级别:512, 1024, 2048，如果myBuffer cap 是600，那么应该放在512的级别
		for i := 0; i < len(pool.classesSize); i++ {
			if pool.classesSize[i] == size {
				pool.classes[i].Put(b)
				return
			}
			if pool.classesSize[i] > size {
				pool.classes[i-1].Put(b)
				return
			}
		}
		//当然可以从级别大到小比较size, 但是不是符合CPU 加载cacheline习惯
	}
}
