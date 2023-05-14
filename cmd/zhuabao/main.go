package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

// 抓包，保存到文件，为了避免文件太大，就用轮询保存到不同的文件里，然后删除老的文件。
//tcpdump -i lan tcp port 61954 and host 172.21.193.66 -s64 -c 250000 -w test.pcap
//./main tcpdump -i lan tcp port 61954 and host 172.21.193.66 -s64 -c 250000
func main() {
	size := 2
	files := make([]string, size)
	index := 0
	args := os.Args[2:]
	for {
		filename := time.Now().Format("2006-01-02_15_04_05") + ".pcap"
		//_, err := exec.Command("sh", "-c ", fmt.Sprintf("%s -w %s", os.Args[1], filename)).CombinedOutput()
		_, err := exec.Command(os.Args[1], append(args, "-w", filename)...).CombinedOutput()
		if err != nil {
			log.Println(err)
			return
		}

		//替换文件名前先删除
		if file := files[index%size]; file != "" {
			output, err := exec.Command("rm", file).CombinedOutput()
			if err != nil {
				log.Println(err)
				log.Printf("output:%s\n", string(output))
				return
			}
			log.Printf("rm %s ok\n", file)
		}
		files[index%size] = filename
		index++
	}
}
