package server

import (
	"errors"

	"github.com/jursonmo/practise/pkg/proto/session"
)

func (c *Server) AddRouter(msgid uint16, r session.Router) error {
	//业务数据id 从1开始，0 预留给了心跳报文
	if msgid == HeartBeatId {
		return errors.New("msgid must gt 0")
	}
	return c.addRouter(msgid, r)
}

func (s *Server) addRouter(msgid uint16, r session.Router) error {
	return s.routers.AddRouter(msgid, r)
}

func (s *Server) GetRouter(msgid uint16) session.Router {
	return s.routers.GetRouter(msgid)
}
