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

	ttl      time.Duration
	respChan chan HbPkg
	startSeq uint32
	onFlyReq HbPkg
	failCnt  int
	err      error

	requestHandler func(req HbPkg) error
	onSuccess      func(ttl time.Duration)
	onFail         func()

	Config
}
type HbOption func(*Heartbeat)

func WithFailHandler(f func()) HbOption {
	return func(h *Heartbeat) {
		h.onFail = f
	}
}

func WithSuccessHandler(f func(time.Duration)) HbOption {
	return func(h *Heartbeat) {
		h.onSuccess = f
	}
}

func WithStartSeq(start uint32) HbOption {
	return func(h *Heartbeat) {
		h.startSeq = start
	}
}

func NewHeartbeart(c Config, sendRequest func(req HbPkg) error, opts ...HbOption) *Heartbeat {
	if sendRequest == nil {
		return nil
	}
	hb := &Heartbeat{Config: c, requestHandler: sendRequest, respChan: make(chan HbPkg, 2)}
	hb.startSeq = rand.Uint32()
	for _, opt := range opts {
		opt(hb)
	}
	//init request seq
	hb.onFlyReq.Seq = hb.startSeq
	return hb
}

func (hb *Heartbeat) RecvResp(p HbPkg) {
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
func (hb *Heartbeat) Run() error {
	if hb.err != nil {
		return hb.err
	}

	timer := NewTimerx(hb.Intvl)
	defer timer.Stop()

	for {
		//always check ctx first
		select {
		case <-hb.ctx.Done():
			hb.err = hb.ctx.Err()
			return hb.err
		default:
		}

		select {
		case p := <-hb.respChan:
			if p.Seq != hb.onFlyReq.Seq {
				continue
			}
			hb.ttl = timex.Since(p.Ts)
			if hb.onSuccess != nil {
				hb.onSuccess(hb.ttl)
			}
			//ok, reset
			hb.failCnt = 0
			timer.Reset(hb.Intvl)

		case <-timer.Done():
			if err := hb.timeout(); err != nil {
				return err
			}
			hb.onFlyReq.Seq++
			hb.onFlyReq.Ts = timex.Now()
			err := hb.requestHandler(hb.onFlyReq)
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
func (hb *Heartbeat) timeout() error {
	if hb.err != nil {
		return hb.err
	}

	hb.failCnt++
	if hb.failCnt <= hb.Probes {
		return nil
	}

	//fail
	hb.err = fmt.Errorf("heartbeat stop, timeout cnt:%d", hb.failCnt)
	if hb.onFail != nil {
		hb.onFail()
	}
	return hb.err
}