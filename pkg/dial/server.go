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

type Server struct {
	ctx         context.Context
	cancel      context.CancelFunc
	handler     func(net.Conn) error
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

func WithHandler(handler func(net.Conn) error) ServerOption {
	return func(s *Server) {
		s.handler = handler
	}
}

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

			s.tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
			s.tlsConf.ClientCAs = clientCertPool
		}
	}

	if s.tlsConf != nil && s.tlsConf.CipherSuites == nil {
		//s.tlsConf.CipherSuites = NO_DES
		s.tlsConf.CipherSuites = SecureCipherSuites()
	}
	return s, nil
}

func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	for _, ln := range s.lns {
		ln.Close()
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

	accpet := func(index int, ln net.Listener, endpoint *url.URL) error {
		log.Printf("server(%d) start and listen at %s://%s\n", index, endpoint.Scheme, endpoint.Host)
		log.Printf("index:%d, ln.Addr():%s://%s\n", index, ln.Addr().Network(), ln.Addr().String())
		defer log.Printf("index:%d, ln.Addr():%s://%s out service\n", index, ln.Addr().Network(), ln.Addr().String())
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
			go s.handler(conn)
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
	lc.Control = TcpUserTimeoutControl(userTimeout)
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
