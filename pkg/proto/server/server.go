package server

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/jursonmo/practise/pkg/dial"
	"github.com/jursonmo/practise/pkg/proto"
	"github.com/jursonmo/practise/pkg/proto/session"
)

const (
	HeartBeatId = 0
)

var ErrInvalidData = errors.New("invalid data")

type Server struct {
	sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	closed bool

	server *dial.Server

	routers *session.RouterRegister

	//handlers
	onConnect func(session.Sessioner)
	onStop    func(session.Sessioner)
}

type Option func(*Server)

func NewServer(endpoints []string, opts ...Option) (*Server, error) {
	var err error
	s := &Server{routers: session.NewRouterRegister()}
	s.server, err = dial.NewServer(endpoints, dial.WithHandler(s.connHandle))
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Server) connHandle(conn net.Conn, listener_id int) error {
	log.Printf("new conn:%v->%v", conn.LocalAddr(), conn.RemoteAddr())
	pconn := proto.NewProtoConn(conn, true, nil)
	err := pconn.Init(s.ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	session := NewSession(s, pconn)
	go session.Start(s.ctx)

	log.Println("session start")
	if s.onConnect != nil {
		s.onConnect(session)
	}
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s.server.Start(s.ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	s.Lock()
	if s.closed {
		s.Unlock()
		return errors.New("already closed")
	}
	s.closed = true
	s.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
