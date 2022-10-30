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
	handler     func(net.Conn) error
	keepalive   time.Duration
	userTimeout time.Duration
	lis         net.Listener
	tlsConf     *tls.Config
	certFile    string
	keyFile     string
	clientCert  string
	endpoint    *url.URL
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

func NewServer(addr string, options ...ServerOption) (*Server, error) {
	s := &Server{}
	for _, opt := range options {
		opt(s)
	}
	endpoint, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	s.endpoint = endpoint

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
		s.tlsConf.CipherSuites = NO_DES
	}
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	nctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.ctx = nctx

	endpoint := s.endpoint
	l, err := NewListener("tcp4", endpoint.Host, s.keepalive, s.userTimeout)
	if err != nil {
		return err
	}
	switch endpoint.Scheme {
	case "tls":
		//s.lis, err = tls.Listen("tcp4", endpoint.Host, s.tlsConf)
		s.lis, err = tlsListen(l, s.tlsConf)
		if err != nil {
			return err
		}
	case "tcp":
		//s.lis, err = net.Listen("tcp4", endpoint.Host)
		s.lis = l
	}

	log.Printf("server start and listen at %s://%s\n", endpoint.Scheme, endpoint.Host)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		conn, err := s.lis.Accept()
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

func NewListener(network, laddr string, keepalive, userTimeout time.Duration) (net.Listener, error) {
	var lc net.ListenConfig
	lc.KeepAlive = keepalive
	lc.Control = TcpUserTimeoutControl(userTimeout)
	l, err := lc.Listen(context.Background(), network, laddr)
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
