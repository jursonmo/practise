package server

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

// Config is for configuring server
type Config struct {
	Address    string `toml:"address"`
	TLSEnable  bool   `toml:"tls_enable"`
	CertFile   string `toml:"cert_file"`
	KeyFile    string `toml:"key_file"`
	ClientCert string `toml:"client_cert"`
}

type server struct {
	address    string
	tgPath     string
	auth       map[string]string
	tlsEnable  bool
	certFile   string
	keyFile    string
	clientCert string

	httpSrv *http.Server
}

func newServer(cfg *Config) *server {
	srv := &server{
		address:    cfg.Address,
		certFile:   cfg.CertFile,
		keyFile:    cfg.KeyFile,
		tlsEnable:  cfg.TLSEnable,
		clientCert: cfg.ClientCert,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("xxx", handlerx)
	srv.httpSrv = &http.Server{Addr: cfg.Address, Handler: mux}
	return srv
}

func handlerx(http.ResponseWriter, *http.Request) {

}

func (s *server) start() error {
	if !s.tlsEnable {
		return s.httpSrv.ListenAndServe()
	}

	tlsConfig := new(tls.Config)
	if s.clientCert != "" {
		pool := x509.NewCertPool()
		caCrt, err := ioutil.ReadFile(s.clientCert)
		if err != nil {

		} else {
			pool.AppendCertsFromPEM(caCrt)
			tlsConfig.ClientCAs = pool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	s.httpSrv.TLSConfig = tlsConfig

	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		log.Printf("LoadX509KeyPair server cert and key at %q and %q: %v. Use self-signed certificate",
			s.certFile, s.keyFile, err)
		// cert, err = GenCertificate()
		// if err != nil {
		// 	log.Printf("gen tls.Certificate: %v, use http", err)
		// 	log.Printf("server serve at %v/http", s.httpSrv.Addr)
		// 	s.httpSrv.ListenAndServe()
		// }
		return err
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	log.Printf("server serve at %v/https\n", s.httpSrv.Addr)
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return err
	}

	defer ln.Close()
	kal := tcpKeepAliveListener{ln.(*net.TCPListener)}
	return s.httpSrv.Serve(tls.NewListener(kal, tlsConfig))

}

func (s *server) stop() error {
	// s.httpSrv.Shutdown(context.TODO())
	return s.httpSrv.Close()
}

//wrap 包裹一个net.TCPListener，然后重写Accept()方法，加上自定义的代码。
type tcpKeepAliveListener struct{ *net.TCPListener }

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
