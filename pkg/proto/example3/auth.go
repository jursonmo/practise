package main

type AuthReq struct {
	User string
	Pwd  string
}

var UserName = "tom"
var Password = "123456"
var authReq = AuthReq{User: UserName, Pwd: Password}
