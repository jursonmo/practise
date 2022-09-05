package singletask

import "time"

type Backoffer interface {
	Duration() time.Duration
	Reset()
}

type defaultBackoff struct{}

func (b *defaultBackoff) Duration() time.Duration {
	return time.Second * 5
}
func (b *defaultBackoff) Reset() {}

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
