package dial

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func ReuseportControl(fs ...func(network, address string, c syscall.RawConn) error) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		for _, f := range fs {
			if err := f(network, address, c); err != nil {
				return err
			}
		}

		var syscallErr error
		controlErr := c.Control(func(fd uintptr) {
			syscallErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		})
		if syscallErr != nil {
			return syscallErr
		}
		return controlErr
	}
}
