package backoffhandle

import (
	"context"
	"errors"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

type BackoffHandle struct {
	handle       func(context.Context) error
	resultHandle func(error)
	backoff      backoffx.Backoffer
}

func NewBackoffHandle(handle func(context.Context) error, backoff backoffx.Backoffer, resultHandle func(error)) *BackoffHandle {
	return &BackoffHandle{handle: handle, backoff: backoff, resultHandle: resultHandle}
}

func (bh *BackoffHandle) Run(ctx context.Context) error {
	var start time.Time
	first := true
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !first {
			DelayCtx(ctx, start, bh.backoff.Duration())
		}
		first = false
		err := bh.handle(ctx)
		if err == nil {
			bh.backoff.Reset()
			return err
		}
		if bh.resultHandle != nil {
			bh.resultHandle(err)
		}
		if IsContextErrs(err) {
			return err
		}
	}
}

func IsContextErrs(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return false
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
