package singletask

import (
	"context"
	"errors"
	"time"
)

type MustSuccess struct {
	ctx context.Context
	err error

	f        func() error
	intvl    time.Duration //at least interval time to call f
	quitErrs []error       //if f() return one of quitErrs, don't try to call f
}

func ContextErrs() []error {
	return []error{context.Canceled, context.DeadlineExceeded}
}

var DefMustSuccess = NewMustSuccess(context.Background(), time.Second*5, ContextErrs())

func NewMustSuccess(ctx context.Context, intvl time.Duration, errs []error) *MustSuccess {
	return &MustSuccess{ctx: ctx, intvl: intvl, quitErrs: errs}
}

func (ms *MustSuccess) Call(f func(context.Context) error, resultHandlers ...TaskResultHandler) error {
	if ms.Error() != nil {
		return ms.Error()
	}
	for {
		if err := ms.ctx.Err(); err != nil {
			return err
		}
		start := time.Now()
		err := f(ms.ctx)
		for _, handler := range resultHandlers {
			handler(err)
		}
		if err == nil {
			return nil
		}

		for _, quitErr := range ms.quitErrs {
			if errors.Is(err, quitErr) {
				ms.err = err
				return ms.err
			}
		}
		DelayCtx(ms.ctx, start, ms.intvl)
	}
}

func (ms *MustSuccess) Error() error {
	return ms.err
}

func (ms *MustSuccess) Reset(ctx context.Context, intvl time.Duration) {
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
