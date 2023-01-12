package trytosucce

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

var DefMustSuccess = NewMustSuccess(context.Background(), time.Second*5, []error{context.Canceled, context.DeadlineExceeded})

func NewMustSuccess(ctx context.Context, intvl time.Duration, errs []error) *MustSuccess {
	return &MustSuccess{ctx: ctx, intvl: intvl, quitErrs: errs}
}

type ErrorHandler func(error)

func (ms *MustSuccess) Call(f func(context.Context) error, errorHandlers ...ErrorHandler) error {
	if ms.Error() != nil {
		return ms.Error()
	}
	for {
		if err := ms.ctx.Err(); err != nil {
			return err
		}
		start := time.Now()
		err := f(ms.ctx)
		if err == nil {
			return nil
		}
		for _, errorHandler := range errorHandlers {
			errorHandler(err)
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

func (ms *MustSuccess) Reset(ctx context.Context) {
	ms.ctx = ctx
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
