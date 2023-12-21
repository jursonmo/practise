package main

import (
	"fmt"
	"time"
)

var timeformat = "20060102150405"
var timeformatNoyear = "0102150405"
var timeformatMs = "20060102150405.999" //必须是.999才能打印毫秒
var timeFmtWithMS = "2006-01-02 15:04:05.999"

func main() {
	now := time.Now()
	fmt.Println("Println:", now)
	fmt.Printf("2006-01-02 15:04:05, format:%s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("%s, format:%s\n", timeformat, now.Format(timeformat))

	fmt.Printf("%s, UTC format:%s\n", timeformat, now.UTC().Format(timeformat))

	fmt.Printf("%s, no year format:%s\n", timeformatNoyear, now.Format(timeformatNoyear))
	fmt.Printf("%s, with ms format:%s\n", timeformatMs, now.Format(timeformatMs))
	fmt.Printf("%s, with ms format:%s\n", timeFmtWithMS, now.Format(timeFmtWithMS))

}
