package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
)

type Func func(ctx context.Context) (context.Context, error)

// Named wraps a Func with a human-readable name for diagnostic logging.
// When used with AllConcurrent, the name will appear in progress logs.
func Named(name string, fn Func) Func {
	return func(ctx context.Context) (context.Context, error) {
		start := time.Now()
		_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "[setup] %s: starting\n", name)
		ctx, err := fn(ctx)
		elapsed := time.Since(start).Truncate(time.Millisecond)
		if err != nil {
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "[setup] %s: FAILED after %s: %v\n", name, elapsed, err)
		} else {
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "[setup] %s: done (%s)\n", name, elapsed)
		}
		return ctx, err
	}
}
