package setup

import (
	"context"
	"fmt"
	"io"
	"time"

	e2econtext "github.com/loft-sh/e2e-framework/pkg/context"
	"github.com/onsi/ginkgo/v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

func All(fns ...Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		for _, fn := range fns {
			var err error
			ctx, err = fn(ctx)
			if err != nil {
				return ctx, err
			}
		}
		return ctx, nil
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
