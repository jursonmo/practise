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
