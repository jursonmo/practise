package main

import (
	"fmt"
	"net/url"
)

func main() {
	u, err := url.Parse("https://user:password@www.baidu.com:443/api?location=shenzhen&mail=xxx@163.com")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", u.String())
	fmt.Printf("%#v\n", u)
	fmt.Printf("%#v\n", u.User)
	//test()
}

/*
go run url.go
https://user:password@www.baidu.com:443/api?location=shenzhen&mail=xxx@163.com
&url.URL{Scheme:"https", Opaque:"", User:(*url.Userinfo)(0xc000064180), Host:"www.baidu.com:443", Path:"/api", RawPath:"", ForceQuery:false, RawQuery:"location=shenzhen&mail=xxx@163.com", Fragment:"", RawFragment:""}
&url.Userinfo{username:"user", password:"password", passwordSet:true}
*/

type A struct {
	b *B
}
type B struct {
	Name string
}

func (a *A) GetName() string {
	return a.b.GetName()
}
func (b *B) GetName() string {
	if b == nil {
		return "null"
	}
	return b.Name
}

func test() {
	a := (*A)(nil)
	fmt.Println(a.GetName())
}
