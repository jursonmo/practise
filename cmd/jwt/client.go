package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

/*
一次典型的访问场景是：
浏览器发送http请求（没有Authorization header）
服务器端返回401页面
浏览器弹出认证对话框
用户输入帐号密码，并点确认
浏览器再次发出http请求（带着Authorization header）
服务器端认证通过，并返回页面
浏览器显示页面
*/
func main() {
	// 登录请求
	loginData := url.Values{
		"username": {"your_username"},
		"password": {"your_password"},
	}
	resp, err := http.PostForm("http://localhost:8080/login", loginData)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 获取JWT令牌
	tokenBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println("token:", token)
	// 请求受限资源
	req, err := http.NewRequest("GET", "http://localhost:8080/restricted", nil)
	if err != nil {
		log.Fatal(err)
	}

	// 在请求头中添加JWT令牌
	req.Header.Set("Authorization", token)

	// 发送请求
	client := http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 解析响应
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	body := string(bodyBytes)

	fmt.Println(resp.Status)
	fmt.Println(body)
}
