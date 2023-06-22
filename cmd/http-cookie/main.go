package main

import (
	"fmt"
	"log"
	"net/http"
)

/*
举个例子，下面是一个server端的响应：

HTTP/2.0 200 OK
Content-Type: text/html
Set-Cookie: name=flydean
Set-Cookie: site=www.flydean.com
当浏览器接收到这个响应之后，就会在本地的cookies中设置对应的值，并且在后续的请求中将这些值以cookies的header形式带上：

GET /test.html HTTP/2.0
Host: www.flydean.com
Cookie: name=flydean; site=www.flydean.com
*/
func main() {
	// 设置登录路由
	http.HandleFunc("/login", loginHandler)

	// 设置业务路由
	http.HandleFunc("/business", businessHandler)

	// 启动服务器
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// 在登录处理程序中，生成一个Cookie并将其设置在响应中
	cookie := &http.Cookie{
		Name:  "session",
		Value: "business1", // 这里假设登录成功后将业务标识存储在Cookie中
		Path:  "/",
	}
	http.SetCookie(w, cookie)

	// 登录成功后重定向到业务页面
	http.Redirect(w, r, "/business", http.StatusSeeOther)
}

func businessHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求中获取Cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		// 如果没有找到Cookie，则表示用户未登录
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 根据Cookie的值判断业务类型
	business := cookie.Value

	// 根据业务类型进行相应的处理
	switch business {
	case "business1":
		fmt.Fprintf(w, "Welcome to Business 1")
	case "business2":
		fmt.Fprintf(w, "Welcome to Business 2")
	default:
		fmt.Fprintf(w, "Unknown Business")
	}
}
