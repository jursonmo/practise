package backoffhandle

import (
	"context"
	"errors"
	"time"

	"github.com/jursonmo/practise/pkg/backoffx"
)

// 保证至少间隔一定的时间后再重试,避免过于频繁调用下一个服务
// 不能用sleep, 因为这样可能间隔的时间超过预想的，handle 处理会花费时间，同时sleep期间无法被唤醒或cancel,
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
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		start = time.Now()
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
		//time.Sleep(bh.backoff.Duration() - time.Since(start)) ??
		DelayCtx(ctx, start, bh.backoff.Duration()) //如果ctx 已经deadline了，DelayCtx也会快速返回
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
	defer cancel()
	<-ctx.Done()
}
