package session

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
)

const (
	HeartBeatReqId  = 0
	HeartBeatRespId = 1
	MaxPrivateId    = 10
)

type Sessioner interface {
	Name() string
	SessionID() string
	UnderlayConn() net.Conn
	Endpoints() []*url.URL
	WriteMsg(uint16, []byte) error
}

type BaseSession struct{}

var ErrNonImplement = errors.New("non implement")

func (bs *BaseSession) Name() string {
	return "non implement"
}

func (bs *BaseSession) SessionID() string {
	return "non implement"
}

func (bs *BaseSession) UnderlayConn() net.Conn {
	return nil
}
func (bs *BaseSession) Endpoints() []*url.URL {
	return nil
}
func (bs *BaseSession) WriteMsg(id uint16, d []byte) error {
	return ErrNonImplement
}

type Router interface {
	Handle(Sessioner, uint16, []byte)
}

type HandleFunc func(Sessioner, uint16, []byte)

func (h HandleFunc) Handle(s Sessioner, id uint16, d []byte) {
	h(s, id, d)
}

type RouterRegister struct {
	sync.RWMutex
	routers map[uint16]Router
}

func NewRouterRegister() *RouterRegister {
	return &RouterRegister{routers: make(map[uint16]Router)}
}

func (rr *RouterRegister) AddRouter(id uint16, r Router) error {
	rr.Lock()
	defer rr.Unlock()
	rr.routers[id] = r
	return nil
}

func (rr *RouterRegister) GetRouter(id uint16) Router {
	rr.RLock()
	defer rr.RUnlock()
	return rr.routers[id]
}

var ErrMsgId error

func init() {
	ErrMsgId = fmt.Errorf("msgid must gt MaxPrivateId:%d", MaxPrivateId)

}
func CheckMsgId(msgid uint16) error {
	//业务数据id 从10开始，0-9 预留给了心跳报文
	if msgid < MaxPrivateId {
		return ErrMsgId
	}
	return nil
}
