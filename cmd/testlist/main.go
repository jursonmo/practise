package main

import (
	"fmt"

	"github.com/jursonmo/practise/cmd/testlist/ilist"
)

type node struct {
	ilist.Entry
	index int
}

func main() {
	list1 := ilist.List{}
	for i := 0; i < 10; i++ {
		n := &node{index: i}
		list1.PushBack(n)
	}
	list2 := list1
	for !list2.Empty() {
		e := list2.Front()
		list2.Remove(e)
		n := e.(*node)
		fmt.Printf("list2 node index:%d\n", n.index)
	}
	fmt.Printf("list1 empty:%v\n", list1.Empty())
	for !list1.Empty() {
		e := list1.Front()
		list1.Remove(e)
		n := e.(*node)
		fmt.Printf("list1 node index:%d\n", n.index)
	}
}
