package singletask

import (
	"context"
	"errors"
	"time"
)

type Promise struct {
	ctx context.Context
	err error

	f        func() error
	intvl    time.Duration //at least interval time to call f
	quitErrs []error       //if f() return one of quitErrs, don't try to call f
}

func ContextErrs() []error {
	return []error{context.Canceled, context.DeadlineExceeded}
}

var DefPromise = NewPromise(context.Background(), time.Second*5, ContextErrs())

func NewPromise(ctx context.Context, intvl time.Duration, errs []error) *Promise {
	return &Promise{ctx: ctx, intvl: intvl, quitErrs: errs}
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
			return ms
		}

		for _, quitErr := range ms.quitErrs {
			if errors.Is(err, quitErr) {
				ms.err = err
				return ms
			}
		}
		DelayCtx(ms.ctx, start, ms.intvl)
	}
}

func (ms *Promise) Error() error {
	return ms.err
}

func (ms *Promise) Reset(ctx context.Context, intvl time.Duration) {
	ms.ctx = ctx
	ms.intvl = intvl
	ms.err = nil
}

func DelayCtx(ctx context.Context, start time.Time, delayAtLeast time.Duration) {
	cost := time.Since(start)
	if cost >= delayAtLeast {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, delayAtLeast-cost)
	<-ctx.Done()
	cancel()
}
