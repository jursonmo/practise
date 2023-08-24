//go:build !linux
// +build !linux

package dial

import "syscall"

func ReuseportControl(fs ...func(network, address string, c syscall.RawConn) error) func(network, address string, c syscall.RawConn) error {
	return nil
}
