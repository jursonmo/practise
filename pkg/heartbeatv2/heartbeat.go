package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jursonmo/practise/pkg/retrystrategy"
	"github.com/jursonmo/practise/pkg/timex"
)

// net.ipv4.tcp_keepalive_time=7200
// net.ipv4.tcp_keepalive_intvl=75
// net.ipv4.tcp_keepalive_probes=9
// TCP_KEEPCNT                                 = 0x6
// TCP_KEEPIDLE                                = 0x4
// TCP_KEEPINTVL                               = 0x5
const (
	REQUEST  = 0
	RESPONSE = 1
)

type HbPkg struct {
	T   byte //type
	Seq uint32
	Ts  time.Duration
}

func (hbp *HbPkg) IsResponse() bool {
	return hbp.T == RESPONSE
}

func (hbp *HbPkg) IsRequest() bool {
	return hbp.T == REQUEST
}

func (hbp *HbPkg) GenResponse() []byte {
	if !hbp.IsRequest() {
		return nil
	}
	hbp.T = RESPONSE
	d, _ := json.Marshal(hbp)
	return d
}

type Heartbeat struct {
	ctx context.Context

	name     string
	rs       retrystrategy.RetryStrategyer
	rrt      time.Duration //心跳的rrt round-trip time
	pktChan  chan HbPkg
	startSeq uint32
	onFlyReq HbPkg
	err      error

	send       func(req []byte) error
	onResponse func(name string, ttl time.Duration)  //收到心跳回应是的回调
	onTimeout  func(name string, dead time.Duration) //dead 表示死了多久，即多久没有收到心跳

}
type HbOption func(*Heartbeat)

func WithOnTimout(f func(string, time.Duration)) HbOption {
	return func(h *Heartbeat) {
		h.onTimeout = f
	}
}

func WithResponseHandler(f func(string, time.Duration)) HbOption {
	return func(h *Heartbeat) {
		h.onResponse = f
	}
}

func WithStartSeq(start uint32) HbOption {
	return func(h *Heartbeat) {
		h.startSeq = start
	}
}

// retrystrategy.DefaultHbRetryStrategyer
// send 函数用来发送请求，同时可以用来发送回应数据。
func NewHeartbeat(name string, rs retrystrategy.RetryStrategyer, send func([]byte) error, opts ...HbOption) *Heartbeat {
	if send == nil {
		return nil
	}
	hb := &Heartbeat{name: name, rs: rs, send: send, pktChan: make(chan HbPkg, 2)}
	hb.startSeq = rand.Uint32()
	for _, opt := range opts {
		opt(hb)
	}
	//init request seq
	hb.onFlyReq.Seq = hb.startSeq
	return hb
}

func (hb *Heartbeat) PutHbData(d []byte) error {
	p := HbPkg{}
	err := json.Unmarshal(d, &p)
	if err != nil {
		return err
	}
	select {
	case hb.pktChan <- p:
	default:
		return fmt.Errorf("pktChan full and drop hb HbPkg:%v", p)
	}
	return nil
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

// 1. NewHeartbeart(config), with OnFail, with OnSuccess
// 2. hb.RecvResp() -->channel
// 3. hb.Run --> recvResp, call OnSuccess if recvRespon succesfully, call OnFail
func (hb *Heartbeat) Start(ctx context.Context) error {
	if hb.err != nil {
		return hb.err
	}
	hb.ctx = ctx
	timer := NewTimerx(hb.rs.Duration())
	defer timer.Stop()

	//一开始先发请求
	hb.sendRequest()

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
		case p := <-hb.pktChan:
			if p.IsRequest() {
				//回应心跳请求
				data := p.GenResponse()
				hb.send(data)
				continue
			}
			//handle response
			if p.Seq != hb.onFlyReq.Seq {
				continue
			}
			//ok, reset
			hb.rs.Reset()                 //重置“重试策略”
			timer.Reset(hb.rs.Duration()) //确保定时器重置，重置的时间是由“重试策略”决定的重试间隔。
			hb.rrt = timex.Since(p.Ts)
			if hb.onResponse != nil {
				hb.onResponse(hb.name, hb.rrt)
			}
		case <-timer.Done():
			//这里表示心跳超时
			if !hb.rs.Retryable() {
				if hb.onTimeout != nil {
					hb.onTimeout(hb.name, hb.rs.RetryTime())
				}
				hb.err = fmt.Errorf("hb:%s timeout:%v, tried %d times", hb.name, hb.rs.RetryTime(), hb.rs.Tried())
				return hb.err
			}
			//定期器到了, 发送心跳请求
			hb.sendRequest()
			timer.Reset(hb.rs.Duration())
		}
	}
}

func (hb *Heartbeat) sendRequest() error {
	hb.onFlyReq.T = REQUEST
	hb.onFlyReq.Seq++
	hb.onFlyReq.Ts = timex.Now()
	d, err := json.Marshal(&hb.onFlyReq)
	if err != nil {
		hb.err = err
		return hb.err
	}
	err = hb.send(d)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (hb *Heartbeat) IsFail() bool {
	return hb.err != nil
}
