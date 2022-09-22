package retrystrategy

import (
	"sync"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

type RetryStrategyer interface {
	Retryable() bool
	Tried() int //how many time have tried
	MaxRetry() int
	backoffx.Backoffer
}

var _ RetryStrategyer = (*RetryStrategy)(nil)

type RetryStrategy struct {
	sync.Mutex
	maxRetry int
	retry    int
	backoff  backoffx.Backoffer
}

//maxRetry == 0 means not limit
func NewRetryStrategy(maxRetry int, backoff backoffx.Backoffer) *RetryStrategy {
	return &RetryStrategy{maxRetry: maxRetry, backoff: backoff}
}

func (r *RetryStrategy) Retryable() bool {
	r.Lock()
	defer r.Unlock()
	if r.maxRetry > 0 && r.retry >= r.maxRetry {
		return false
	}
	return true
}

//next Duration
func (r *RetryStrategy) Duration() time.Duration {
	r.Lock()
	defer r.Unlock()
	r.retry++
	return r.backoff.Duration()
}

func (r *RetryStrategy) Reset() {
	r.Lock()
	defer r.Unlock()
	r.retry = 0
	r.backoff.Reset()
}

func (r *RetryStrategy) Tried() int {
	r.Lock()
	defer r.Unlock()
	return r.retry
}

func (r *RetryStrategy) MaxRetry() int {
	return r.maxRetry
}

/*
----------------------------------
----------------------------------
*/
//HeartbeatBackoff 类似于tcp keepalive 的做法，正常情况下以intvl 的时间间隔发送心跳,
//如果没有收到心跳回应，那么就要在较短的时间间隔failIntvl内再次发送心跳
//如果收到心跳回应，就重置时间间隔，以intvl的时间间隔发送心跳，否则继续以failIntvl间隔发送心跳,
//连续多次收不到心跳回应，就认为网络断开。
//这样做的目的是不想心跳报文占用过多的网络资源，同时又能快速探测出网络是否断开
type HeartbeatBackoff struct {
	intvl     time.Duration
	failIntvl time.Duration // smaller than intvl
	failRetry bool
}

//第一次获取next Duration, 返回正常情况下的间隔时间
func (hb *HeartbeatBackoff) Duration() time.Duration {
	if hb.failRetry {
		return hb.failIntvl
	}
	hb.failRetry = true
	return hb.intvl
}

//每次收到心跳回应后，都要调用reset(), 然后再调用Duration()来获取下次发送心跳的间隔时间。
func (hb *HeartbeatBackoff) Reset() {
	hb.failRetry = false
}

func NewHeartbeatBackoff(intvl, failIntvl time.Duration) backoffx.Backoffer {
	return &HeartbeatBackoff{intvl: intvl, failIntvl: failIntvl}
}
