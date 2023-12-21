package main

import (
	"fmt"
	"log"
	"os"
)

func initLogFile() {
	logFile, err := os.OpenFile("./123.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
		return
	}
	log.SetOutput(logFile)
	log.SetPrefix("[这个是日志的固定前缀]]")
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	log.Printf("打印到终端\n")
	initLogFile()
	fmt.Println("________")
	log.Println("人生")
	fmt.Println("_________+")
}
