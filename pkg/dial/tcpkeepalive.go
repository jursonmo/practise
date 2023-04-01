//go:build !linux
// +build !linux

package dial

import "net"

func SetKeepaliveParameters(conn net.Conn, idle, intvl, cnt int) error {
	return nil
}
