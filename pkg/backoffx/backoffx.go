package backoffx

import (
	"time"

	"github.com/rfyiamcool/backoff"
)

type Backoffer interface {
	Duration() time.Duration
	Reset()
}

var DefaultBackoff Backoffer = &LinearBackoff{time.Second * 5}

type LinearBackoff struct {
	d time.Duration
}

func (rb *LinearBackoff) Duration() time.Duration {
	return rb.d
}

func (rb *LinearBackoff) Reset() {}

func NewLinearBackoff(d time.Duration) Backoffer {
	return &LinearBackoff{d: d}
}

//github.com/rfyiamcool/backoff
func NewDynamicBackoff(minDuration, maxDuration time.Duration, factor float64) Backoffer {
	return backoff.NewBackOff(
		backoff.WithMinDelay(minDuration),
		backoff.WithMaxDelay(maxDuration),
		backoff.WithFactor(factor), //1.5
	)
}
