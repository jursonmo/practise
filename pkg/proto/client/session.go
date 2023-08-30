package client

import (
	"fmt"
	"net"
	"net/url"

	"github.com/jursonmo/practise/pkg/proto"
	"github.com/jursonmo/practise/pkg/proto/session"
	"golang.org/x/sync/errgroup"
)

type Session struct {
	session.BaseSession
	id   string
	name string
	//srv  *Server
	cli *Client
	pc  *proto.ProtoConn
	eg  *errgroup.Group

	//routers *session.RouterRegister
}

func NewSession(cli *Client, pc *proto.ProtoConn) *Session {
	return &Session{cli: cli, pc: pc}
}

//实现session.Sessioner接口
func (s *Session) Name() string {
	if s.name != "" {
		return s.name
	}
	if conn := s.UnderlayConn(); conn != nil {
		return fmt.Sprintf("%v->%v", conn.LocalAddr(), conn.RemoteAddr())
	}
	return ""
}
func (s *Session) UnderlayConn() net.Conn {
	return s.pc.Conn()
}

func (s *Session) SessionID() string {
	if s.id != "" {
		return s.id
	}
	if conn := s.UnderlayConn(); conn != nil {
		return fmt.Sprintf("%v->%v", conn.LocalAddr(), conn.RemoteAddr())
	}
	return ""
}
func (s *Session) Endpoints() []*url.URL {
	return s.cli.Endpoints()
}
func (s *Session) WriteMsg(msgid uint16, d []byte) error {
	_, err := s.pc.WriteWithId(msgid, d)
	return err
}
