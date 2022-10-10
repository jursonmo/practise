// +build !linux

package dial

import (
	"syscall"
	"time"
)

func keepaliveControl(t time.Duration) func(network, address string, c syscall.RawConn) error {
	return nil
}
