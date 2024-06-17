package main

import (
	"fmt"

	"github.com/jursonmo/practise/pkg/pool"
)

func assert(b bool) {
	if !b {
		panic("")
	}
}

func main() {
	p := pool.NewSyncPool(128, 4096, 2) //[128, 256, 512, 1024, 2048, 4096]
	//最小级别
	targetSize := 127
	b := p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 128)
	assert(p.Free(b))

	targetSize = 128
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 128)
	assert(p.Free(b))

	targetSize = 129
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 256)
	assert(p.Free(b))

	targetSize = 600
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 1024)
	assert(p.Free(b))

	targetSize = 1023
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 1024)
	assert(p.Free(b))

	targetSize = 1024
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 1024)
	assert(p.Free(b))

	targetSize = 1025
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(cap(b) == 2048)
	assert(p.Free(b))

	//最大值
	targetSize = 4096
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(p.Free(b))

	//大于最大值
	targetSize = 4097
	b = p.Alloc(targetSize)
	fmt.Printf("targe size:%d, get buf len:%d, cap:%d\n", targetSize, len(b), cap(b))
	assert(!p.Free(b)) //超过最大级别，不能放入pool
}
