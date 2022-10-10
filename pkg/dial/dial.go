package dial

import (
	"context"
	"crypto/tls"
	"math"
	"net"
	"net/url"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
	"github.com/rfyiamcool/backoff"
)

var NO_DES = []uint16{
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
}

func SleepWithCtx(ctx context.Context, start time.Time, maxSleep time.Duration) {
	cost := time.Since(start)
	if cost >= maxSleep {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, maxSleep-cost)
	defer cancel()
	<-ctx.Done()
}

// type BackOffer interface {
// 	Duration() time.Duration
// 	Reset()
// }

type DialConfig struct {
	Timeout      time.Duration
	KeepAlive    time.Duration //for linux: use socket option: tcp_user_timeout
	BackOff      backoffx.Backoffer
	MaxDial      int64
	DialFailFunc func(error)
	//Dial(network, address string) (net.Conn, error)
}

func NewDefDialConfig() *DialConfig {
	return &DialConfig{
		Timeout: 3 * time.Second,
		BackOff: backoff.NewBackOff(backoff.WithMinDelay(2*time.Second), backoff.WithMaxDelay(10*time.Second)),
		MaxDial: math.MaxInt64, //default dial forever until ctx cancel or timeout
	}
}

/*
go build main.go
# github.com/jursonmo/practise/pkg/dial
../dial.go:69:34: undefined: unix.TCP_USER_TIMEOUT
xxx-MBP:example obc$ GOOS=linux go build main.go
*/
// var control = func(network, address string, c syscall.RawConn) error {
// 	var syscallErr error
// 	controlErr := c.Control(func(fd uintptr) {
// 		syscallErr = syscall.SetsockoptInt(
// 			int(fd), syscall.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, 5*1000)
// 	})
// 	if syscallErr != nil {
// 		return syscallErr
// 	}
// 	if controlErr != nil {
// 		return controlErr
// 	}
// 	return nil
// }

type DialOption func(cc *DialConfig)

func WithBackOffer(bo backoffx.Backoffer) DialOption {
	return func(c *DialConfig) {
		c.BackOff = bo
	}
}

func WithTimeout(t time.Duration) DialOption {
	return func(c *DialConfig) {
		c.Timeout = t
	}
}
func WithKeepAlive(t time.Duration) DialOption {
	return func(c *DialConfig) {
		c.KeepAlive = t
	}
}

func WithMaxDial(n int64) DialOption {
	return func(c *DialConfig) {
		c.MaxDial = n
	}
}

func WithDialFailFunc(f func(error)) DialOption {
	return func(c *DialConfig) {
		c.DialFailFunc = f
	}
}

//dial until success or ctx error
func Dial(ctx context.Context, addr string, options ...DialOption) (conn net.Conn, err error) {
	c := NewDefDialConfig()
	for _, opt := range options {
		opt(c)
	}
	network, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	type dialContexter interface {
		DialContext(ctx context.Context, network, address string) (net.Conn, error)
	}
	var dialer dialContexter

	control := keepaliveControl(c.KeepAlive)
	switch network.Scheme {
	case "tls":
		tlsconf := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS11,
			CipherSuites:       NO_DES,
		}
		//conn, err = tls.Dial("tcp", network.Host, tlsconf)
		//conn, err = tls.DialWithDialer(&net.Dialer{Timeout: c.Timeout}, "tcp", network.Host, tlsconf)
		td := &tls.Dialer{NetDialer: &net.Dialer{Timeout: c.Timeout, Control: control}, Config: tlsconf}
		//conn, err = td.DialContext(ctx, "tcp", network.Host)
		dialer = td
	case "tcp":
		d := &net.Dialer{Timeout: c.Timeout, Control: control}
		//conn, err = d.DialContext(ctx, "tcp", network.Host)
		dialer = d
	}

	for i := 0; i < int(c.MaxDial); i++ {
		conn, err = dialer.DialContext(ctx, "tcp", network.Host)
		if err == nil {
			c.BackOff.Reset()
			break
		}
		if c.DialFailFunc != nil {
			c.DialFailFunc(err)
		}
		// if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded){
		// 	return nil, err
		// }
		if ctx.Err() != nil {
			return nil, err
		}
		//time.Sleep(c.BackOff.Duration())
		SleepWithCtx(ctx, time.Now(), c.BackOff.Duration())
	}
	return
}
