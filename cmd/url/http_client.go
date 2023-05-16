package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

var Test = "3"

func main() {
	//总结:URL.RawQuery 就如字面意思，原始请求参数，不加任何处理，所以参数有特殊字符，需要encode 后再赋值给RawQuery。服务器那边默认会Decode来获取参数。
	client := &http.Client{}
	var req *http.Request
	var err error

	switch Test {
	case "1":
		//req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/testquery?name=@莫&hobby=#xx", nil)
		//有字符#，服务器那边解析不出来hobby 的值,  服务器打印 req.Form: map[hobby:[] name:[@莫]]

		/*
			//url 参数特殊字符
			+    URL 中+号表示空格                      %2B
			空格 URL中的空格可以用+号或者编码           %20
			/   分隔目录和子目录                        %2F
			?    分隔实际的URL和参数                    %3F
			%    指定特殊字符                           %25
			#    表示书签                               %23
			&    URL 中指定的参数间的分隔符             %26
			=    URL 中指定参数的值                     %3D
		*/
		//把#改成%23, 看服务器能否解析出, 结果也能解析出来
		req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/testquery?name=@莫&hobby=%23xx", nil)
		if err != nil {
			panic(err)
		}
		fmt.Printf("11111, req.URL.RawQuery:%v\n", req.URL.RawQuery) //11111, req.URL.RawQuery:name=@莫&hobby=%23xx
		//抓包看, testquery?name=@\350\216\253&hobby=%23xx, 只是把‘莫’ unicode,
	case "2":
		req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/testquery", nil)
		if err != nil {
			panic(err)
		}
		parm := make(url.Values)
		parm.Add("name", "@莫")
		parm.Add("hobby", "#xx")
		//URL.RawQuery 就如字面意思，原始请求参数，不加任何处理，所以参数有特殊字符，需要encode 后再赋值给RawQuery。服务器那边默认会Decode来获取参数。
		req.URL.RawQuery = parm.Encode()                             //经过url encode 后，服务器那边才正常解析出参数hobby, 打印：req.Form: map[hobby:[#xx] name:[@莫]]
		fmt.Printf("22222, req.URL.RawQuery:%v\n", req.URL.RawQuery) //22222, req.URL.RawQuery:hobby=%23xx&name=%40%E8%8E%AB
	case "3":
		parm := make(url.Values)
		parm.Add("name", "@莫")
		parm.Add("hobby", "#xx")
		encodeQuery := parm.Encode()
		fmt.Printf("parm.Encode():%v\n", parm.Encode()) //hobby=%23xx&name=%40%E8%8E%AB,  即用%表示经过了编码

		req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:1313/testquery"+"?"+encodeQuery, nil) //这样也可以
		if err != nil {
			panic(err)
		}
		fmt.Printf("3333, req.URL.RawQuery:%v\n", req.URL.RawQuery)
	}
	//抓包看，2，3 两个case 的query都是hobby=%23xx&name=%40%E8%8E%AB, 只是传入的方式不一样，
	// case 2 是我手动直接赋值 req.URL.RawQuery, case 3 是让NewRequest 把? 问号后面的query 赋值给req.URL.RawQuery
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(data))
}

/*
总结：req.URL.RawQuery 必须是不包含特殊字符的，如果有就自己编码。
1. ? 问号后面的query 赋值给req.URL.RawQuery， 所以如果用？来传递参数，且参数包含特殊字符时，需要自己编码；
2. 最好是手动设置 req.URL.RawQuery = parm.Encode()
*/
