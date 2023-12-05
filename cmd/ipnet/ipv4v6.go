package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	iptype, err := CheckIpType("2001:0db8:86a3:08d3:1319:8ae:0370:7344/128")
	if err != nil {
		panic(err)
	}
	fmt.Println(iptype)
}

func CheckIpType(addr string) (string, error) {
	var ip net.IP
	var err error
	if strings.Contains(addr, "/") {
		ip, _, err = net.ParseCIDR(addr)
		if err != nil {
			return "", fmt.Errorf("net.ParseCIDR addr:%v err:%v", addr, err)
		}
	} else {
		ip = net.ParseIP(addr)
	}
	if ip.To4() != nil {
		return "ipv4", nil
	} else if ip.To16() != nil {
		return "ipv6", nil
	}
	return "", fmt.Errorf("unknown address type, addr:%s", addr)
}
