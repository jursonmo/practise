package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/jursonmo/practise/pkg/heartbeat"
	"github.com/jursonmo/practise/pkg/proto"
	"github.com/jursonmo/practise/pkg/proto/session"
	"golang.org/x/sync/errgroup"
)

type Session struct {
	session.BaseSession
	id   string
	name string
	srv  *Server
	pc   *proto.ProtoConn
	eg   *errgroup.Group

	routers *session.RouterRegister
}

func NewSession(s *Server, pc *proto.ProtoConn) *Session {
	ss := &Session{srv: s, pc: pc, name: "a session from server", routers: session.NewRouterRegister()}
	pc.SetMsgHandler(proto.ProtoMsgHandle(ss.msgHandle))
	//设置了SetMsgHandlerv2, 上面设置的SetMsgHandler 就不起作用了
	pc.SetMsgHandlerv2(proto.ProtoMsgHandlev2(ss.msgHandlev2))
	return ss
}

func (s *Session) String() string {
	return fmt.Sprintf("name:%s, id:%s, %v", s.Name(), s.SessionID(), s.pc)
}

func (s *Session) addRouter(id uint16, r session.Router) {
	s.routers.AddRouter(id, r)
}

func (s *Session) GetRouter(id uint16) session.Router {
	return s.routers.GetRouter(id)
}

func (s *Session) msgHandle(pc *proto.ProtoConn, d []byte, t byte) error {
	if len(d) < 3 {
		return ErrInvalidData
	}
	msgid := binary.BigEndian.Uint16(d)

	//主要是 session 内部注册的私有数据处理，比如心跳处理
	if r := s.GetRouter(msgid); r != nil {
		r.Handle(s, msgid, d[2:])
		return nil
	}

	//用户注册的msg router
	r := s.srv.GetRouter(msgid)
	if r == nil {
		return nil
	}
	r.Handle(s, msgid, d[2:])
	return nil
}

func (s *Session) msgHandlev2(pc *proto.ProtoConn, pkg proto.Pkger) error {
	msgid, ok := pkg.MsgId()
	if !ok {
		log.Println("session msgHandlev2 can't get msgid")
		return nil
	}
	//log.Printf("session get msgid:%d", msgid)
	//主要是 session 内部注册的私有数据处理，比如心跳处理
	if r := s.GetRouter(msgid); r != nil {
		r.Handle(s, msgid, pkg.Paylaod())
		return nil
	}

	//用户注册的msg router
	r := s.srv.GetRouter(msgid)
	if r == nil {
		return nil
	}
	r.Handle(s, msgid, pkg.Paylaod())
	return nil
}

func (s *Session) Start(ctx context.Context) error {
	var egctx context.Context
	s.eg, egctx = errgroup.WithContext(ctx) //要用c.ctx, 这样c.cancel 才能 取消egctx
	s.eg.Go(func() error {
		err := s.pc.Run(egctx)
		log.Println(err)
		return err
	})

	//注册heartbeat 处理请求回调，默认就是回应原始数据,类型是HeartBeatRespId
	s.addRouter(uint16(session.HeartBeatReqId),
		session.HandleFunc(func(s session.Sessioner, msgid uint16, d []byte) {
			log.Printf("receive hb request:%s", string(d))
			err := s.WriteMsg(session.HeartBeatRespId, d)
			if err != nil {
				log.Println(err)
			}
		}))

	//发送心跳
	hbsend := func(req heartbeat.HbPkg) error {
		buf := bytes.NewBuffer(make([]byte, 0, 128))
		//binary.Write(buf, binary.BigEndian, uint16(session.HeartBeatReqId))
		encoder := json.NewEncoder(buf)
		err := encoder.Encode(&req)
		if err != nil {
			return err
		}
		log.Printf("send heartbeat req:%+v\n", req)
		//_, err = s.pc.Write(buf.Bytes())
		_, err = s.pc.WriteWithId(session.HeartBeatReqId, buf.Bytes())
		return err
	}

	heartbeater := heartbeat.NewHeartbeat(s.name,
		heartbeat.DefautConfig, hbsend)

	//注册心跳回应处理
	s.addRouter(uint16(session.HeartBeatRespId), session.HandleFunc(func(s session.Sessioner, id uint16, d []byte) {
		hb := heartbeat.HbPkg{}
		err := json.Unmarshal(d, &hb)
		if err != nil {
			log.Println("handle hb response err:", err)
			return
		}
		log.Printf("receive hb response:%+v\n", hb)
		heartbeater.PutResponse(hb)
	}))

	s.eg.Go(func() error {
		err := heartbeater.Start(egctx)
		log.Printf("heartbeat quit, err:%v", err)
		return err
	})

	return nil
}

//实现session.Sessioner接口
func (s *Session) UnderlayConn() net.Conn {
	return s.pc.Conn()
}
func (s *Session) SessionID() string {
	return s.id
}

func (s *Session) WriteMsg(msgid uint16, d []byte) error {
	// 这里需要make 一个大的内存对象，还需要copy一次
	// buf := make([]byte, len(d)+2)
	// binary.BigEndian.PutUint16(buf, msgid)
	// copy(buf[2:], d)
	// _, err := s.pc.Write(buf)
	// return err

	return s.WriteMsgv2(msgid, d)
}

func (s *Session) WriteMsgv2(msgid uint16, d []byte) error {
	_, err := s.pc.WriteWithId(msgid, d)
	return err
}
func (s *Session) Stop() {
	log.Printf("session:%v, stopping...", s)
	if s.srv.onStop != nil {
		s.srv.onStop(s)
	}
	s.eg.Wait()
	log.Printf("session:%v, stoped", s)
}
