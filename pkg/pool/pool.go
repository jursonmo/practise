//https://github.com/TerryMarszr0/GoServer/blob/master/src/common/pool/sync_pool.go
/***********************************************************************
* @ 临时对象池
* @ brief
	1、我们可以把sync.Pool类型值看作是存放可被重复使用的值的容器，自动伸缩、高效、并发安全

	2、它会专门为每一个与操作它的goroutine相关联的Pool都生成一个本地池。

	3、在临时对象池的Get方法被调用的时候，它一般会先尝试从与本地Pool对应的那个本地池中获取一个对象值。
		如果获取失败，它就会试图从其他Pool的本地池中偷一个对象值并直接返回给调用方。
		如果依然未果，那它只能把希望寄托于当前的临时对象池的New字段代表的那个对象值生成函数了。
		注意，这个对象值生成函数产生的对象值永远不会被放置到池中。它会被直接返回给调用方。

	4、临时对象池的Put方法会把它的参数值存放到与当前P对应的那个本地池中。
		每个P的本地池中的绝大多数对象值都是被同一个临时对象池中的所有本地池所共享的。也就是说，它们随时可能会被偷走

	5、对gc友好，gc执行时临时对象池中的某个对象值仅被该池引用，那么它可能会在gc时被回收

	6、原生的sync.Pool有个较大的问题：我们不能自由控制Pool中元素的数量，放进Pool中的对象每次GC发生时可能都会被清理掉

* @ author 达达
* @ date 2016-7-23
************************************************************************/
package pool

import "sync"

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
// pool := NewSyncPool(128, 1024, 2)
func NewSyncPool(minSize, maxSize, factor int) *SyncPool {
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
		pool.classes[n].New = func(size int) func() interface{} { //为唯一公开字段New赋值
			return func() interface{} {
				return make([]byte, size)
			}
		}(chunkSize)
		n++
	}
	return pool
}

// add by mo: 二分查找，避免pool级别比较多时，查找的次数太多了，特别是获取最小的buf时。
func binarySearchBetween(arr []int, target int) int {
	left, right := 0, len(arr)-1

	//获取最小级别的buf, 直接返回，不需要二分查找了
	if target <= arr[0] {
		return 0
	}

	// 当数组中只有一个的话，肯定不满足需求了; 由于在调用binarySearchBetween之前，有判断target是否小于最大大值,所以这里不可能发生
	if len(arr) < 2 {
		return -1
	}

	for left <= right {
		mid := left + (right-left)/2

		// 检查目标值是否在 arr[mid-1] 和 arr[mid] 之间
		if mid > 0 && arr[mid-1] < target && target <= arr[mid] {
			return mid
		}

		// 根据目标值调整左右边界
		if target < arr[mid] {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return -1 // 未找到符合条件的位置
}

// Alloc try alloc a []byte from internal slab class if no free chunk in slab class Alloc will make one.
func (self *SyncPool) Alloc(size int) []byte {
	if size <= self.maxSize {
		// for i := 0; i < len(self.classesSize); i++ {
		// 	if self.classesSize[i] >= size {
		// 		mem := self.classes[i].Get().([]byte) //sync.Pool.Get()返回interface{}
		// 		return mem[:size]
		// 	}
		// }
		// 上面的代码，如果级别比较多，查找的次数太多了，特别是获取最小的buf时。

		i := binarySearchBetween(self.classesSize, size)
		if i != -1 {
			mem := self.classes[i].Get().([]byte)
			return mem[:size]
		}

	}
	return make([]byte, size)
}

// Free release a []byte that alloc from Pool.Alloc.
func (self *SyncPool) Free(mem []byte) bool {
	if size := cap(mem); size <= self.maxSize {
		// for i := 0; i < len(self.classesSize); i++ {
		// 	if self.classesSize[i] >= size {
		// 		self.classes[i].Put(mem)
		// 		return
		// 	}
		// }
		i := binarySearchBetween(self.classesSize, size)
		if i != -1 {
			self.classes[i].Put(mem)
			return true
		}
	}
	return false
}
