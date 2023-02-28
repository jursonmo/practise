package main

import (
	"container/heap"
	"fmt"
)

type item struct {
	v uint64
}

var topK = 10

type topitems []item

func (top topitems) Len() int {
	return len(top)
}

// 绑定Less方法，这里用的是小于号，生成的是小根堆
func (top topitems) Less(i, j int) bool {
	return top[i].v < top[j].v
}

// 绑定swap方法
func (top topitems) Swap(i, j int) {
	top[i], top[j] = top[j], top[i]
}

// 绑定put方法，
func (top *topitems) Pop() interface{} {
	old := *top
	n := len(old)
	item := old[n-1]
	*top = old[0 : n-1]
	return item
}

// 绑定push方法
func (top *topitems) Push(x interface{}) {
	*top = append(*top, x.(item))
}

func (top *topitems) CanPush() bool {
	return len(*top) < topK
}

// topK Push
func (top *topitems) topKPush(i item) {
	if top.CanPush() {
		heap.Push(top, i)
		//fmt.Println("push", top, i)
		return
	}
	if (*top)[0].v > i.v {
		//fmt.Printf("(*top)[0].v:%d,  i.v:%d \n", (*top)[0].v, i.v)
		return
	}
	//fmt.Println(top, "try add ", i)
	(*top)[0] = i
	heap.Fix(top, 0)
	//fmt.Println("fix:", top)
}

func main() {
	var topitem topitems //=topitems{item{8}, item{3}}
	// heap.Init(&topitem)
	// fmt.Println("init:", topitem)

	testData := []item{item{4}, item{7}, item{2}, item{5}, item{9}, item{6}}
	topK = 4

	for _, v := range testData {
		(&topitem).topKPush(v)
	}

	fmt.Printf("topK:%d -------\n", topK)
	for len(topitem) > 0 {
		item := heap.Pop(&topitem).(item)
		fmt.Printf("item.v :%d\n", item.v)
	}
}

/*
output:
topK:4 -------
item.v :5
item.v :6
item.v :7
item.v :9
*/
