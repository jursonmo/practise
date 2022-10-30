package dial

import (
	"golang.org/x/sys/unix"
	"syscall"
	"time"
)

//golang.org/x/sys v0.0.0-20210902050250-f475640dd07b
func TcpUserTimeoutControl(t time.Duration) func(network, address string, c syscall.RawConn) error {
	if t < time.Millisecond {
		return nil
	}
	return func(network, address string, c syscall.RawConn) error {
		var syscallErr error
		controlErr := c.Control(func(fd uintptr) {
			syscallErr = syscall.SetsockoptInt(
				int(fd), syscall.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, int(t.Milliseconds()))
		})
		if syscallErr != nil {
			return syscallErr
		}
		if controlErr != nil {
			return controlErr
		}
		return nil
	}
}
