package dial

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"time"
)

type ServerConfig struct {
	Vid         int
	Local       bool
	ListenAddrs []string
}

type ConnHandler func(conn net.Conn, fromLnID int) error
type Server struct {
	ctx         context.Context
	cancel      context.CancelFunc
	handler     ConnHandler
	keepalive   time.Duration
	userTimeout time.Duration
	lns         []net.Listener
	tlsConf     *tls.Config
	certFile    string
	keyFile     string
	clientCert  string
	endpoints   []*url.URL
}

type ServerOption func(s *Server)

func ServerKeepalive(t time.Duration) ServerOption {
	return func(s *Server) {
		s.keepalive = t
	}
}

func ServerUserTimeout(t time.Duration) ServerOption {
	return func(s *Server) {
		s.userTimeout = t
	}
}

func WithHandler(handler ConnHandler) ServerOption {
	return func(s *Server) {
		s.handler = handler
	}
}

// 底层都是tcp listener, 如果是tls, 用原始tcp listener 和tlsConfig 生成新的tls listener: tls.NewListener(l, tlsConfig), 同样是net.Listener
func NewServer(addrs []string, options ...ServerOption) (*Server, error) {
	s := &Server{}
	for _, opt := range options {
		opt(s)
	}
	for _, addr := range addrs {
		endpoint, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		s.endpoints = append(s.endpoints, endpoint)
	}

	if s.tlsConf == nil {
		if s.certFile != "" && s.keyFile != "" {
			cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
			if err != nil {
				return nil, err
			}
			s.tlsConf = new(tls.Config)
			s.tlsConf.Certificates = []tls.Certificate{cert}
		}

		if s.clientCert != "" {
			certBytes, err := ioutil.ReadFile(s.clientCert)
			if err != nil {
				return nil, err
			}
			clientCertPool := x509.NewCertPool()
			ok := clientCertPool.AppendCertsFromPEM(certBytes)
			if !ok {
				return nil, fmt.Errorf("AppendCertsFromPEM err")
			}
			if s.tlsConf == nil {
				s.tlsConf = new(tls.Config)
			}
			s.tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
			s.tlsConf.ClientCAs = clientCertPool
		}
	}

	if s.tlsConf != nil && s.tlsConf.CipherSuites == nil {
		//s.tlsConf.CipherSuites = NO_DES //go早起版本，tlsConf.CipherSuites 默认会包含DES 这种不安全加密方式。所以这里指定“NO_DES”
		s.tlsConf.CipherSuites = SecureCipherSuites() //tls.InsecureCipherSuites() 包含不安全的加密套件, 可以查看这些不安全的套件是哪些
	}
	return s, nil
}

func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	for _, ln := range s.lns {
		if ln != nil {
			ln.Close()
		}
	}
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	nctx, cancel := context.WithCancel(context.Background())
	s.ctx = nctx
	s.cancel = cancel

	s.lns = make([]net.Listener, len(s.endpoints))
	for i, endpoint := range s.endpoints {
		l, err := NewListener(s.ctx, "tcp4", endpoint.Host, s.keepalive, s.userTimeout)
		if err != nil {
			return err
		}
		switch endpoint.Scheme {
		case "tls":
			//s.lis, err = tls.Listen("tcp4", endpoint.Host, s.tlsConf)
			s.lns[i], err = tlsListen(l, s.tlsConf)
			if err != nil {
				return err
			}
		case "tcp":
			//s.lis, err = net.Listen("tcp4", endpoint.Host)
			s.lns[i] = l
		}
	}

	accpet := func(lnID int, ln net.Listener, endpoint *url.URL) error {
		log.Printf("server(%d) start and listen at %s://%s\n", lnID, endpoint.Scheme, endpoint.Host)
		log.Printf("lnID:%d, ln.Addr():%s://%s\n", lnID, ln.Addr().Network(), ln.Addr().String())
		defer log.Printf("lnID:%d, ln.Addr():%s://%s out service\n", lnID, ln.Addr().Network(), ln.Addr().String())
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			//ln.Close make ln.Accept() return
			conn, err := ln.Accept()
			if err, ok := err.(net.Error); ok && err.Temporary() {
				time.Sleep(time.Millisecond * 500)
				continue
			}
			if err != nil {
				return err
			}
			go s.handler(conn, lnID)
		}
	}

	for i, ln := range s.lns {
		go accpet(i, ln, s.endpoints[i])
	}
	return nil
}

func NewListener(ctx context.Context, network, laddr string, keepalive, userTimeout time.Duration) (net.Listener, error) {
	var lc net.ListenConfig
	lc.KeepAlive = keepalive
	//lc.Control 是一个函数指针，想对fd设置多个属性，那就wrap 包装下
	lc.Control = TcpUserTimeoutControl(userTimeout, ReuseportControl())
	l, err := lc.Listen(ctx, network, laddr)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func tlsListen(l net.Listener, config *tls.Config) (net.Listener, error) {
	if config == nil || len(config.Certificates) == 0 &&
		config.GetCertificate == nil && config.GetConfigForClient == nil {
		return nil, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}
	return tls.NewListener(l, config), nil
}
