package httpu

import (
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// MultiClient dispatches requests out to all the delegated clients.
type MultiClient struct {
	// The HTTPU clients to delegate to.
	delegates []ClientInterface
}

var _ ClientInterface = &MultiClient{}

// NewMultiClient creates a new MultiClient that delegates to all the given
// clients.
func NewMultiClient(delegates []ClientInterface) *MultiClient {
	return &MultiClient{
		delegates: delegates,
	}
}

// Do implements ClientInterface.Do.
func (mc *MultiClient) Do(
	req *http.Request,
	timeout time.Duration,
	numSends int,
) ([]*http.Response, error) {
	tasks := &errgroup.Group{}

	results := make(chan []*http.Response)
	tasks.Go(func() error {
		defer close(results)
		return mc.sendRequests(results, req, timeout, numSends)
	})

	var responses []*http.Response
	tasks.Go(func() error {
		for rs := range results {
			responses = append(responses, rs...)
		}
		return nil
	})

	return responses, tasks.Wait()
}

func (mc *MultiClient) sendRequests(
	results chan<- []*http.Response,
	req *http.Request,
	timeout time.Duration,
	numSends int,
) error {
	tasks := &errgroup.Group{}
	for _, d := range mc.delegates {
		d := d // copy for closure
		tasks.Go(func() error {
			responses, err := d.Do(req, timeout, numSends)
			if err != nil {
				return err
			}
			results <- responses
			return nil
		})
	}
	return tasks.Wait()
}

// MultiClientCtx dispatches requests out to all the delegated clients.
type MultiClientCtx struct {
	// The HTTPU clients to delegate to.
	delegates []ClientInterfaceCtx
}

var _ ClientInterfaceCtx = &MultiClientCtx{}

// NewMultiClient creates a new MultiClient that delegates to all the given
// clients.
func NewMultiClientCtx(delegates []ClientInterfaceCtx) *MultiClientCtx {
	return &MultiClientCtx{
		delegates: delegates,
	}
}

// DoWithContext implements ClientInterfaceCtx.DoWithContext.
func (mc *MultiClientCtx) DoWithContext(
	req *http.Request,
	numSends int,
) ([]*http.Response, error) {
	tasks, ctx := errgroup.WithContext(req.Context())
	req = req.WithContext(ctx) // so we cancel if the errgroup errors
	results := make(chan []*http.Response)

	// For each client, send the request to it and collect results.
	tasks.Go(func() error {
		defer close(results)
		return mc.sendRequestsCtx(results, req, numSends)
	})

	var responses []*http.Response
	tasks.Go(func() error {
		for rs := range results {
			responses = append(responses, rs...)
		}
		return nil
	})

	return responses, tasks.Wait()
}

func (mc *MultiClientCtx) sendRequestsCtx(
	results chan<- []*http.Response,
	req *http.Request,
	numSends int,
) error {
	tasks := &errgroup.Group{}
	for _, d := range mc.delegates {
		d := d // copy for closure
		tasks.Go(func() error {
			responses, err := d.DoWithContext(req, numSends)
			if err != nil {
				return err
			}
			results <- responses
			return nil
		})
	}
	return tasks.Wait()
}
