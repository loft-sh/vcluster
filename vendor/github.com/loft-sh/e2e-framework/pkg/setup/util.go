package setup

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	e2econtext "github.com/loft-sh/e2e-framework/pkg/context"
	"github.com/onsi/ginkgo/v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

type retryOptions struct {
	interval time.Duration
	timeout  time.Duration
	writer   io.Writer
}

const (
	DefaultPollingInterval = time.Second * 2
	DefaultPollingTimeout  = time.Second * 20
)

type Option func(*retryOptions)

func ToEnvFunc(fn Func) types.EnvFunc {
	return func(ctx context.Context, config *envconf.Config) (context.Context, error) {
		return fn(ctx)
	}
}

func WithPollingInterval(d time.Duration) Option {
	return func(o *retryOptions) { o.interval = d }
}

func WithPollingTimeout(d time.Duration) Option {
	return func(o *retryOptions) { o.timeout = d }
}

func WithWriter(w io.Writer) Option {
	return func(o *retryOptions) {
		o.writer = w
	}
}

type key int

const (
	resultsKey key = iota
)

type Result struct {
	Context context.Context
	Err     error
}

func ResultsFrom(ctx context.Context) []Result {
	if value := ctx.Value(resultsKey); value != nil {
		return value.([]Result)
	}
	return nil
}

func WithResults(ctx context.Context, results []Result) context.Context {
	return context.WithValue(ctx, resultsKey, results)
}

func All(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		var errs []error
		for _, fn := range fns {
			var err error
			if ctx, err = fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return ctx, errors.NewAggregate(errs)
	}
}

func AllFailFast(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		for _, fn := range fns {
			var err error
			if ctx, err = fn(ctx); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	}
}

func AllWithResults(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		var results []Result
		var errs []error
		for _, fn := range fns {
			var err error
			ctx, err = fn(ctx)
			errs = append(errs, err)
			results = append(results, Result{
				Context: ctx,
				Err:     err,
			})
		}

		return WithResults(ctx, results), errors.NewAggregate(errs)
	}
}

func AllConcurrent(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		resultsChan := make(chan Result, len(fns))

		wg := new(sync.WaitGroup)
		wg.Add(len(fns))

		for _, fn := range fns {
			go func(setupFn Func) {
				defer wg.Done()
				ctx, err := setupFn(ctx)
				resultsChan <- Result{
					Context: ctx,
					Err:     err,
				}
			}(fn)
		}

		wg.Wait()
		close(resultsChan)

		var errs []error
		for res := range resultsChan {
			ctx = e2econtext.WithValues(ctx, res.Context)
			errs = append(errs, res.Err)
		}

		return ctx, errors.NewAggregate(errs)
	}
}

func AllConcurrentWithResults(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		resultsChan := make(chan Result, len(fns))

		wg := new(sync.WaitGroup)
		wg.Add(len(fns))

		for _, fn := range fns {
			go func(setupFn Func) {
				defer wg.Done()
				ctx, err := setupFn(ctx)
				resultsChan <- Result{
					Context: ctx,
					Err:     err,
				}
			}(fn)
		}

		wg.Wait()
		close(resultsChan)

		var errs []error
		var results []Result
		for res := range resultsChan {
			results = append(results, res)
			errs = append(errs, res.Err)
		}

		return WithResults(ctx, results), errors.NewAggregate(errs)
	}
}

func AsCleanup(fn Func) func(ctx context.Context) func(ctx context.Context) error {
	return func(curr context.Context) func(specContext context.Context) error {
		return func(specContext context.Context) error {
			_, err := fn(e2econtext.WithValues(specContext, curr))
			return err
		}
	}
}

func RetryOnConflict(ctx context.Context, fn Func, opts ...Option) error {
	options := retryOptions{
		interval: DefaultPollingInterval,
		timeout:  DefaultPollingTimeout,
		writer:   ginkgo.GinkgoWriter,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return wait.PollUntilContextTimeout(ctx, options.interval, options.timeout, true, func(ctx context.Context) (done bool, err error) {
		newCtx, err := fn(ctx)
		if kerrors.IsConflict(err) {
			_, _ = fmt.Fprintf(options.writer, "update conflict, retrying: %s\n", err)
			return false, nil
		}
		if err != nil {
			return false, err
		}
		ctx = newCtx
		return true, nil
	})

}
