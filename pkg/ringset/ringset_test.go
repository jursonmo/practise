package ringset

import (
	"fmt"
	"testing"
)

type MyInt int

func (i MyInt) Key() string {
	return fmt.Sprintf("key_%d", i)
}

func TestWriteAndRead(t *testing.T) {
	size := 10
	r := New(size, WithoutMutex(true), WithIsSet(true))

	//1. wirte a element that not implement Key() string, i shoul return error
	_, err := r.Write([]interface{}{int(1)})
	if err == nil {
		t.Fatal("wirte a element that not implement Key() string, i shoul return error")
	}

	//2. write 10 element
	elements := make([]interface{}, 0, size)
	for i := 0; i < 10; i++ {
		elements = append(elements, MyInt(i))
	}
	n, err := r.Write(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//3. ring is full, write a element will be error
	n, err = r.Write([]interface{}{MyInt(11)})
	if n != 0 || err == nil {
		t.Fatalf("ring set is full, but n:%d != 0 || err(%v) != nil ", n, err)
	}

	//4. but write repeat elements is ok
	n, err = r.Write(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//5. test Read 5 element one by one
	p := make([]interface{}, 1)
	for i := 0; i < 5; i++ {
		n, err := r.Read(p)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(p) {
			t.Fatalf("n:%d ! = len(p):%d", n, len(p))
		}
		if p[0].(MyInt) != MyInt(i) {
			t.Fatalf("p[0].(MyInt):%d, should be %d", p[0].(MyInt), MyInt(i))
		}
	}

	//6. check  Buffered
	if r.Buffered() != 5 {
		t.Fatalf("r.Buffered():%d, should be %d", r.Buffered(), 5)
	}

	//7. test Read 5 element once
	p = make([]interface{}, 5)
	n, err = r.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(p) {
		t.Fatalf("n:%d ! = len(p):%d", n, len(p))
	}
	for i := 0; i < 5; i++ {
		if p[i].(MyInt) != MyInt(i+5) {
			t.Fatalf("p[%d].(MyInt):%d, should be %d", i, p[i].(MyInt), MyInt(i+5))
		}
	}

	t.Logf("ringset:%v", r)
}

func TestDiscard(t *testing.T) {
	size := 10
	r := New(size, WithoutMutex(true), WithIsSet(true))

	//1. write 10 element
	elements := make([]interface{}, 0, size)
	for i := 0; i < 10; i++ {
		elements = append(elements, MyInt(i))
	}
	n, err := r.Write(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//2. Discard 5 elements
	n, err = r.Discard(5)
	if n != 5 || err != nil {
		t.Fatalf("n(%d) != 5 || err(%v) != nil", n, err)
	}

	//3. Discard 8 elements, actually just discard 5 elements
	n, err = r.Discard(8)
	if n != 5 || err != nil {
		t.Fatalf("n(%d) != 5 || err(%v) != nil", n, err)
	}

	//4. write 10 element agian
	n, err = r.Write(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//5.  check  Buffered
	if r.Buffered() != size {
		t.Fatalf("r.Buffered():%d, should be %d", r.Buffered(), 5)
	}

	n, err = r.Discard(size)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size(%d) || err(%v) != nil", n, size, err)
	}

	t.Logf("ringset:%v", r)
}

func TestWriteRoll(t *testing.T) {
	size := 10
	r := New(size, WithoutMutex(true), WithIsSet(true))

	//1. write 10 element: 0-9
	elements := make([]interface{}, 0, size)
	for i := 0; i < 10; i++ {
		elements = append(elements, MyInt(i))
	}
	n, err := r.Write(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//2. now  ring is full, try to overwrite new 10 element: 10-20
	elements = make([]interface{}, 0, size)
	startElem := 10
	for i := startElem; i < startElem+size; i++ {
		elements = append(elements, MyInt(i))
	}
	n, err = r.WriteRoll(elements)
	if n != size || err != nil {
		t.Fatalf("n(%d) != size || err(%v) != nil", n, err)
	}

	//3. test Read 5 element once
	p := make([]interface{}, size)
	n, err = r.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(p) {
		t.Fatalf("n:%d ! = len(p):%d", n, len(p))
	}
	for i := 0; i < size; i++ {
		if p[i].(MyInt) != MyInt(i+startElem) {
			t.Fatalf("p[%d].(MyInt):%d, should be %d", i, p[i].(MyInt), MyInt(i+startElem))
		}
	}

	//4. print ring
	t.Logf("ringset:%v", r)
}
