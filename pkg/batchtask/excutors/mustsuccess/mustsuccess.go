package mustsuccess

import (
	"context"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

//retry forever until be canceled or do() successfully(return nil error)

/*
type Executor interface {
	Do(ctx context.Context, batch []interface{}) error
	Close()
}
*/
//implement Executor interface
type MustSuccess struct {
	do      func(ctx context.Context, batch []interface{}) error
	err     error
	backoff backoffx.Backoffer
}

func NewMustSuccess(do func(ctx context.Context, batch []interface{}) error, backoff backoffx.Backoffer) *MustSuccess {
	ms := &MustSuccess{do: do}
	ms.backoff = backoff
	if ms.backoff == nil {
		ms.backoff = backoffx.NewDynamicBackoff(time.Second, time.Second*10, 1.5)
	}
	return ms
}

func (ms *MustSuccess) Close() {}

func (ms *MustSuccess) Do(ctx context.Context, batch []interface{}) error {
	for {
		start := time.Now()
		err := ms.do(ctx, batch)
		if err == nil {
			ms.Reset()
			return err
		}

		if IsContextErr(err) {
			return err
		}
		DelayAtLeast(ctx, start, ms.Duration())
	}
}

func (ms *MustSuccess) Duration() time.Duration {
	if ms.backoff != nil {
		return ms.backoff.Duration()
	}

	return time.Second * 5
}

func (ms *MustSuccess) Reset() {
	if ms.backoff != nil {
		ms.backoff.Reset()
	}
}

func IsContextErr(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

func DelayAtLeast(ctx context.Context, start time.Time, delayAtLeast time.Duration) {
	//if 'start' is unset
	none := time.Time{}
	if start == none {
		start = time.Now()
	}

	cost := time.Since(start)
	if cost >= delayAtLeast {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, delayAtLeast-cost)
	<-ctx.Done()
	cancel()
}
