package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"crypto/hmac"
	"crypto/sha512"
)

func main() {

	buf := bytes.NewBuffer(make([]byte, 0, 512))
	buf.WriteString("xxxx")
	h := md5.New()
	h.Write(buf.Bytes())
	cipherStr := h.Sum(nil)
	fmt.Println(hex.EncodeToString(cipherStr))

	cipherStr2 := md5.Sum(buf.Bytes())
	fmt.Printf("%x\n", cipherStr2)

	key := []byte("key")
	mac := hmac.New(sha512.New, key)
	mac.Write(buf.Bytes())
	Mac := mac.Sum(nil)
	fmt.Println("sha512 hmac:", hex.EncodeToString(Mac))
}
