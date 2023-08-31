package server

import (
	"github.com/jursonmo/practise/pkg/proto/session"
)

func (c *Server) AddRouter(msgid uint16, r session.Router) error {
	//业务数据id 从10开始，0-9 预留私有控制消息，比如心跳报文
	if err := session.CheckMsgId(msgid); err != nil {
		return err
	}
	return c.addRouter(msgid, r)
}

func (s *Server) addRouter(msgid uint16, r session.Router) error {
	return s.routers.AddRouter(msgid, r)
}

func (s *Server) GetRouter(msgid uint16) session.Router {
	return s.routers.GetRouter(msgid)
}
