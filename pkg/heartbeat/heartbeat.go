package heartbeat

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jursonmo/practise/pkg/timex"
)

// net.ipv4.tcp_keepalive_time=7200
// net.ipv4.tcp_keepalive_intvl=75
// net.ipv4.tcp_keepalive_probes=9
// TCP_KEEPCNT                                 = 0x6
// TCP_KEEPIDLE                                = 0x4
// TCP_KEEPINTVL                               = 0x5

var Heartbeater interface {
	Done() <-chan time.Time
	Beatting() error
}

var DefautConfig = Config{Intvl: time.Second * 5, IntvlOnFail: time.Second * 2, Probes: 3}

type Config struct {
	Intvl       time.Duration //
	IntvlOnFail time.Duration
	Probes      int
}

type HbPkg struct {
	T   byte //type
	Seq uint32
	Ts  time.Duration
}

type Heartbeat struct {
	ctx context.Context

	name     string
	lastTime time.Duration //上次收到心跳回应的时间点
	rrt      time.Duration //心跳的rrt round-trip time
	respChan chan HbPkg
	startSeq uint32
	onFlyReq HbPkg
	failCnt  int
	err      error

	send      func(req HbPkg) error
	onSuccess func(name string, ttl time.Duration)  //
	onTimeout func(name string, dead time.Duration) //dead 表示死了多久，即多久没有收到心跳

	Config
}
type HbOption func(*Heartbeat)

func WithOnTimout(f func(string, time.Duration)) HbOption {
	return func(h *Heartbeat) {
		h.onTimeout = f
	}
}

func WithSuccessHandler(f func(string, time.Duration)) HbOption {
	return func(h *Heartbeat) {
		h.onSuccess = f
	}
}

func WithStartSeq(start uint32) HbOption {
	return func(h *Heartbeat) {
		h.startSeq = start
	}
}

func NewHeartbeat(name string, c Config, sendRequest func(HbPkg) error, opts ...HbOption) *Heartbeat {
	if sendRequest == nil {
		return nil
	}
	hb := &Heartbeat{name: name, Config: c, send: sendRequest, respChan: make(chan HbPkg, 2)}
	hb.startSeq = rand.Uint32()
	for _, opt := range opts {
		opt(hb)
	}
	//init request seq
	hb.onFlyReq.Seq = hb.startSeq
	return hb
}

func (hb *Heartbeat) PutResponse(p HbPkg) {
	select {
	case hb.respChan <- p:
	default:
		log.Printf("drop hb HbPkg:%v", p)
	}
}

type timerx struct {
	timer *time.Timer
}

func NewTimerx(d time.Duration) *timerx {
	return &timerx{timer: time.NewTimer(d)}
}

func (t *timerx) Reset(d time.Duration) {
	t.timer.Stop()
	t.timer.Reset(d)
}

func (t *timerx) Stop() {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C: // try to drain the channel
		default:
		}
	}
}

func (t *timerx) Done() <-chan time.Time {
	return t.timer.C
}

//1. NewHeartbeart(config), with OnFail, with OnSuccess
//2. hb.RecvResp() -->channel
//3. hb.Run --> recvResp, call OnSuccess if recvRespon succesfully, call OnFail
func (hb *Heartbeat) Start(ctx context.Context) error {
	if hb.err != nil {
		return hb.err
	}
	hb.ctx = ctx
	timer := NewTimerx(hb.Intvl)
	defer timer.Stop()

	for {
		//always check ctx first
		if err := hb.ctx.Err(); err != nil {
			hb.err = err
			return hb.err
		}

		select {
		case <-hb.ctx.Done():
			hb.err = hb.ctx.Err()
			return hb.err
		case p := <-hb.respChan:
			if p.Seq != hb.onFlyReq.Seq {
				continue
			}
			//ok, reset
			hb.failCnt = 0
			hb.lastTime = timex.Now() //记录收到心跳回应的时间
			timer.Reset(hb.Intvl)     //确保定时器重置
			hb.rrt = timex.Since(p.Ts)
			if hb.onSuccess != nil {
				hb.onSuccess(hb.name, hb.rrt)
			}
		case <-timer.Done():
			//这里表示心跳超时
			if err := hb.checkTimeout(); err != nil {
				return err
			}
			hb.onFlyReq.Seq++
			hb.onFlyReq.Ts = timex.Now()
			err := hb.send(hb.onFlyReq)
			if err != nil {
				log.Println(err)
			}
			timer.Reset(hb.IntvlOnFail)
		}
	}
}

func (hb *Heartbeat) IsFail() bool {
	return hb.err != nil
}

//handle timeout case
func (hb *Heartbeat) checkTimeout() error {
	if hb.err != nil {
		return hb.err
	}

	hb.failCnt++
	if hb.failCnt <= hb.Probes {
		return nil
	}

	//fail
	hb.err = fmt.Errorf("heartbeat stop, timeout cnt:%d", hb.failCnt)
	if hb.onTimeout != nil {
		hb.onTimeout(hb.name, timex.Since(hb.lastTime))
	}
	return hb.err
}
