// +build !linux

package dial

import (
	"syscall"
	"time"
)

func TcpUserTimeoutControl(t time.Duration) func(network, address string, c syscall.RawConn) error {
	return nil
}
