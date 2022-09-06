package backoffx

import (
	"time"

	"github.com/rfyiamcool/backoff"
)

type Backoffer interface {
	Duration() time.Duration
	Reset()
}

var DefaultBackoff Backoffer = &RegularBackoff{time.Second * 5}

type RegularBackoff struct {
	d time.Duration
}

func (rb *RegularBackoff) Duration() time.Duration {
	return rb.d
}

func (rb *RegularBackoff) Reset() {}

func NewRegularBackoff(d time.Duration) Backoffer {
	return &RegularBackoff{d: d}
}

//github.com/rfyiamcool/backoff
func NewDynamicBackoff(minDuration, maxDuration time.Duration, factor float64) Backoffer {
	return backoff.NewBackOff(
		backoff.WithMinDelay(minDuration),
		backoff.WithMaxDelay(maxDuration),
		backoff.WithFactor(factor), //1.5
	)
}
