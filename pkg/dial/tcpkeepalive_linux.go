package dial

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"strings"
	"syscall"
	"unsafe"
)

///usr/local/go/src/crypto/tls/conn.go
/*
	type Conn struct {
		// constant
		conn    net.Conn
		....
	}
*/

func ConvertTcpConn(conn net.Conn) *net.TCPConn {
	if conn == nil {
		return nil
	}
	if !strings.Contains(conn.LocalAddr().Network(), "tcp") /* && !strings.Contains(conn.LocalAddr().Network(), "tls")*/ {
		log.Printf("unsupport conn network:%s \n", conn.LocalAddr().Network())
		return nil
	}
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		return tcpconn
	}

	tlsconn, ok2 := conn.(*tls.Conn)
	if !ok2 {
		return nil
	}

	type tc struct {
		underConn net.Conn
	}
	tconn := (*tc)(unsafe.Pointer(tlsconn))
	tcpconn, ok = tconn.underConn.(*net.TCPConn)
	if !ok {
		return nil
	}
	return tcpconn
}

/*
# Idle time
cat /proc/sys/net/ipv4/tcp_keepalive_time
# Retry interval
cat /proc/sys/net/ipv4/tcp_keepalive_intvl
# Ping amount
cat /proc/sys/net/ipv4/tcp_keepalive_probes

在Linux中我们可以通过修改 /etc/sysctl.conf 的全局配置：

net.ipv4.tcp_keepalive_time=7200
net.ipv4.tcp_keepalive_intvl=75
net.ipv4.tcp_keepalive_probes=9

两个文件里对keepalive的定义一样的，用谁都行
/usr/local/go/src/syscall/zerrors_linux_amd64.go:
golang.org/x/sys/unix/zerrors_linux.go:

TCP_KEEPCNT                                 = 0x6
TCP_KEEPIDLE                                = 0x4
TCP_KEEPINTVL                               = 0x5
*/
//Sets additional keepalive parameters.
//Uses new interfaces introduced in Go1.11, which let us get connection's file descriptor,
//without blocking, and therefore without uncontrolled spawning of threads (not goroutines, actual threads).
func SetKeepaliveParameters(conn net.Conn, idle, intvl, cnt int) error {
	if idle <= 0 || intvl <= 0 || cnt <= 0 {
		return errors.New("idl <= 0 || intvl <= 0 ||cnt <= 0")
	}

	tconn := ConvertTcpConn(conn)
	if tconn == nil {
		return errors.New("it not tcp\n")
	}
	rawConn, err := tconn.SyscallConn()
	if err != nil {
		log.Printf("on getting raw connection object for keepalive parameter setting", err.Error())
		return err
	}

	rawConn.Control(
		func(fdPtr uintptr) {
			// got socket file descriptor. Setting parameters.
			fd := int(fdPtr)
			//Number of probes.
			err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, cnt) //unix.TCP_KEEPCNT
			if err != nil {
				log.Printf("on setting keepalive probe count", err.Error())
				return
			}
			//Wait time after an unsuccessful probe.
			err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, intvl) //unix.TCP_KEEPINTVL
			if err != nil {
				log.Printf("on setting keepalive retry interval", err.Error())
				return
			}
			err = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, idle) //unix.TCP_KEEPIDLE
			if err != nil {
				log.Printf("on setting keepalive idel", err.Error())
				return
			}
		})
	return err
}
