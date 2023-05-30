package singletask

import (
	"context"
	"errors"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

//promise: will call f loop until f return nil or ctx canceled
type Promise struct {
	ctx context.Context
	err error

	f        func() error
	backoff  backoffx.Backoffer
	quitErrs []error //if f() return one of quitErrs, don't try to call f
}

func ContextErrs() []error {
	return []error{context.Canceled, context.DeadlineExceeded}
}

func (ms *Promise) IsQuitErr(err error) bool {
	for _, quitErr := range ms.quitErrs {
		if errors.Is(err, quitErr) {
			return true
		}
	}
	return false
}

var DefPromise = NewPromise(context.Background(), backoffx.DefaultBackoff, ContextErrs())

func NewPromise(ctx context.Context, backoff backoffx.Backoffer, errs []error) *Promise {
	return &Promise{ctx: ctx, backoff: backoff, quitErrs: errs}
}

func (ms *Promise) Call(f func(context.Context) error, resultHandlers ...TaskResultHandler) *Promise {
	if ms.Error() != nil {
		return ms
	}
	for {
		if err := ms.ctx.Err(); err != nil {
			ms.err = err
			return ms
		}
		start := time.Now()
		err := f(ms.ctx)
		for _, handler := range resultHandlers {
			handler(err)
		}
		if err == nil {
			ms.backoff.Reset()
			return ms
		}

		if ms.IsQuitErr(err) {
			ms.err = err
			return ms
		}
		DelayCtx(ms.ctx, start, ms.backoff.Duration())
	}
}

func (ms *Promise) Error() error {
	return ms.err
}

func (ms *Promise) Reset(ctx context.Context, backoff backoffx.Backoffer) {
	ms.ctx = ctx
	if backoff != nil {
		ms.backoff = backoff
	} else {
		ms.backoff.Reset()
	}
	ms.err = nil
}

func DelayCtx(ctx context.Context, start time.Time, delayAtLeast time.Duration) {
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
	defer cancel()
	<-ctx.Done()
}
