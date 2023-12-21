package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	pkgerr "github.com/pkg/errors"
)

func main() {
	err := io.EOF
	err = fmt.Errorf("11111 %s, %w", "222", err) //%w 可以添加错误信息, 同时保留原来的err
	fmt.Println(err)
	fmt.Printf("using golang errors :%+v\n", err)
	if errors.Is(err, io.EOF) {
		fmt.Printf("err is io.EOF\n")
	}
	var myerr *os.PathError
	if errors.As(err, &myerr) {
		fmt.Println(myerr)
	}

	testPkgerr()
	testErrStack()
}

func testPkgerr() {
	fmt.Println("-------------testPkgerr-----------")
	err := io.EOF
	err = pkgerr.Wrap(err, "mo add 111")

	if errors.Is(err, io.EOF) {
		fmt.Printf("pkg/errors.Wrap err is io.EOF\n") //经过了pkt/errors Wrap 后，error.Is 依然能判断底层err
	}
	fmt.Println(err)
	fmt.Printf("pkt err stack:%+v\n", err) //%+v 它可以打印堆栈
}

func testErrStack() {
	fmt.Println("-------------testErrStack-----------")
	err := pkgerr.New("new my error")
	fmt.Printf("-----pkgerr.New() stack:%+v\n", err)

	err = pkgerr.WithStack(err)
	fmt.Printf("-----pkgerr.WithStack(err):%+v\n", err)

	err = pkgerr.Wrapf(err, "my error:%s", "test")
	fmt.Printf("-----pkgerr.Wrapf():%+v\n", err)
}
