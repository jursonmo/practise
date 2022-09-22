package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
)

type Foo struct {
	name string
	num  int
}

func finalizer(f *Foo) {
	fmt.Println("a finalizer has run for ", f.name, f.num)
}

var counter int

func MakeFoo(name string) (a_foo *Foo) {
	a_foo = &Foo{name, counter}
	counter++
	runtime.SetFinalizer(a_foo, finalizer)
	return
}

func Bar(round int) {
	f1 := MakeFoo("one" + strconv.Itoa(round))
	f2 := MakeFoo("two" + strconv.Itoa(round))
	if counter > 1 {
		fmt.Println("counter >1, SetFinalizer nil ", f2.name, f2.num)
		runtime.SetFinalizer(f2, nil)
	}
	fmt.Println("f1 is: ", f1.name)
	fmt.Println("f2 is: ", f2.name)
}

func main() {
	for i := 0; i < 3; i++ {
		Bar(i)
		time.Sleep(time.Second)
		runtime.GC()
	}
	fmt.Println("done.")
}
