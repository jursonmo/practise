//go:build !linux
// +build !linux

package dial

import (
	"syscall"
	"time"
)

func TcpUserTimeoutControl(t time.Duration, fs ...func(network, address string, c syscall.RawConn) error) func(network, address string, c syscall.RawConn) error {
	return nil
}
