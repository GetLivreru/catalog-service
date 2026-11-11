package processor

import (
	"context"
	"io"
	"sync"
	"time"
)

type (
	CloserFunc        func() error
	CloserContextFunc = func(ctx context.Context) error
)

func (f CloserFunc) Close() error {
	return f()
}

func NewCloserContextFunc(f CloserContextFunc, ctx context.Context, timeout time.Duration) CloserFunc {
	return func() error {
		closeCtx := ctx
		if timeout > 0 {
			var cancelFunc context.CancelFunc
			closeCtx, cancelFunc = context.WithTimeout(ctx, timeout)
			defer cancelFunc()
		}
		return f(closeCtx)
	}
}

func WatchForShutdown(ctx context.Context, wg *sync.WaitGroup, closer io.Closer) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		_ = closer.Close()
	}()
}

func Wrap(ctx context.Context, wg *sync.WaitGroup, cb func(context.Context)) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		if ctx.Err() != nil {
			return
		}
		cb(ctx)
	}()
}
