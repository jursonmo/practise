package retrystrategy

import (
	"sync"
	"time"

	"github.com/jursonmo/practise/pkg/timex"

	"github.com/jursonmo/practise/pkg/backoffx"
)

type RetryStrategyer interface {
	Retryable() bool
	Tried() int               //how many times have tried
	RetryTime() time.Duration //重试了多久，即距离上次resetAt 多长时间了
	MaxRetry() int
	backoffx.Backoffer
}

var _ RetryStrategyer = (*RetryStrategy)(nil)

type RetryStrategy struct {
	sync.Mutex
	maxRetry int
	retry    int
	resetAt  time.Duration
	backoff  backoffx.Backoffer
}

// maxRetry == 0 means not limit
func NewRetryStrategy(maxRetry int, backoff backoffx.Backoffer) RetryStrategyer {
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

// next Duration
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
	r.resetAt = timex.Now()
	r.backoff.Reset()
}

func (r *RetryStrategy) Tried() int {
	r.Lock()
	defer r.Unlock()
	return r.retry
}

func (r *RetryStrategy) RetryTime() time.Duration {
	if r.resetAt == 0 {
		return r.resetAt
	}
	return timex.Since(r.resetAt)
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
	sync.Mutex //当前用到RetryStrategy是可以不用加锁的，RetryStrategy里已经加锁了，如果HeartbeatBackoff用到别的地方呢？所以这里还是加锁吧
	intvl      time.Duration
	failIntvl  time.Duration // smaller than intvl
	failRetry  bool
}

// 第一次获取next Duration, 返回正常情况下的间隔时间
func (hb *HeartbeatBackoff) Duration() time.Duration {
	hb.Lock()
	defer hb.Unlock()
	if hb.failRetry {
		return hb.failIntvl
	}
	hb.failRetry = true
	return hb.intvl
}

// 每次收到心跳回应后，都要调用reset(), 然后再调用Duration()来获取下次发送心跳的间隔时间。
func (hb *HeartbeatBackoff) Reset() {
	hb.Lock()
	defer hb.Unlock()
	hb.failRetry = false
}

func NewHeartbeatBackoff(intvl, failIntvl time.Duration) backoffx.Backoffer {
	return &HeartbeatBackoff{intvl: intvl, failIntvl: failIntvl}
}

var DefaultHbRetryStrategyer RetryStrategyer

func init() {
	hbBackoff := NewHeartbeatBackoff(5*time.Second, time.Second)
	DefaultHbRetryStrategyer = NewRetryStrategy(3, hbBackoff)
}
