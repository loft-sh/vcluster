package command

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// ComponentRunner runs a single attempt of a long-running component (typically
// a subprocess wrapped by RunCommand). It must return when ctx is done. A nil
// return means the component finished gracefully and should not be restarted.
// Any non-nil return is treated as a crash and counted against the restart
// budget.
type ComponentRunner func(ctx context.Context) error

// RestartPolicy controls how Supervisor restarts a component on failure.
type RestartPolicy struct {
	// MaxFailures is the number of failures permitted within FailureWindow
	// before the supervisor escalates (returns ErrEscalate). 0 disables
	// restart entirely (any failure escalates immediately).
	MaxFailures int

	// FailureWindow is the sliding window over which failures accumulate.
	// Failures older than (now - FailureWindow) are forgotten.
	FailureWindow time.Duration

	// MinBackoff is the wait before the first restart attempt after a
	// failure. Each subsequent failure within FailureWindow doubles the
	// backoff, capped at MaxBackoff.
	MinBackoff time.Duration

	// MaxBackoff caps the exponential backoff between restart attempts.
	MaxBackoff time.Duration
}

// DefaultRestartPolicy is a conservative default: tolerate up to 5 failures in
// 5 minutes with 1s..30s backoff before escalating.
func DefaultRestartPolicy() RestartPolicy {
	return RestartPolicy{
		MaxFailures:   5,
		FailureWindow: 5 * time.Minute,
		MinBackoff:    1 * time.Second,
		MaxBackoff:    30 * time.Second,
	}
}

// ErrEscalate is returned by RunWithRestart when the restart budget is
// exhausted and the caller should propagate the failure (e.g. by exiting the
// container so kubelet's CrashLoopBackoff takes over).
var ErrEscalate = errors.New("supervisor: restart budget exhausted")

// SupervisorMetrics is an optional sink the supervisor calls when restart
// events happen. The default Supervisor uses a no-op metrics implementation;
// callers can plug in a Prometheus-backed one without changing the supervisor
// API.
type SupervisorMetrics interface {
	// ObserveRestart is called once per restart attempt, after the backoff
	// has elapsed but before the runner is invoked.
	ObserveRestart(component string, attempt int, sinceLast time.Duration)

	// ObserveExit is called once each time the runner returns (with err nil
	// for graceful, non-nil for crash). It is also called for the final
	// escalation when the budget is exhausted (with the wrapped error).
	ObserveExit(component string, err error)
}

type noopMetrics struct{}

func (noopMetrics) ObserveRestart(string, int, time.Duration) {}
func (noopMetrics) ObserveExit(string, error)                 {}

// RunWithRestart runs the given component, restarting it on failure according
// to policy. It returns:
//
//   - nil when the runner returns nil (graceful completion).
//   - ctx.Err() when ctx is cancelled while the runner is running or the
//     supervisor is sleeping between attempts.
//   - ErrEscalate (wrapped via fmt.Errorf("...: %w", lastErr, ErrEscalate))
//     when the restart budget is exhausted.
//
// The metrics argument may be nil; it is replaced with a no-op implementation.
func RunWithRestart(ctx context.Context, name string, runner ComponentRunner, policy RestartPolicy, metrics SupervisorMetrics) error {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	s := &Supervisor{
		Name:    name,
		Runner:  runner,
		Policy:  policy,
		Metrics: metrics,
	}
	return s.Run(ctx)
}

// Supervisor wraps a single ComponentRunner with restart-on-failure semantics.
// Construct one per component (apiserver, controller-manager, scheduler, kine).
//
// A Supervisor is single-shot: it returns from Run() when the runner finishes
// gracefully, the context is cancelled, or the restart budget is exhausted.
// It is not safe to call Run concurrently from multiple goroutines.
type Supervisor struct {
	Name    string
	Runner  ComponentRunner
	Policy  RestartPolicy
	Metrics SupervisorMetrics

	// now is overridable for tests so we don't have to wait on real
	// time.Now() to verify FailureWindow accounting.
	now func() time.Time

	// sleep is overridable for tests so we can advance the clock without
	// burning real wall time during backoff.
	sleep func(ctx context.Context, d time.Duration) error

	mu       sync.Mutex
	failures []time.Time // truncated to entries within FailureWindow
}

// Run executes the runner repeatedly until graceful completion, ctx
// cancellation, or budget exhaustion.
func (s *Supervisor) Run(ctx context.Context) error {
	if s.Metrics == nil {
		s.Metrics = noopMetrics{}
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.sleep == nil {
		s.sleep = ctxSleep
	}

	var (
		attempt int
		lastErr error
	)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Backoff before retry (skip on first attempt).
		if attempt > 0 {
			backoff := s.computeBackoff(attempt)
			if err := s.sleep(ctx, backoff); err != nil {
				return err
			}
			s.Metrics.ObserveRestart(s.Name, attempt, backoff)
			klog.InfoS("Restarting component", "component", s.Name, "attempt", attempt, "backoff", backoff, "lastErr", lastErr)
		}

		err := s.Runner(ctx)
		s.Metrics.ObserveExit(s.Name, err)

		// Distinguish three terminal cases:
		// 1. Graceful exit (err == nil) — caller signalled completion.
		// 2. Context cancellation — return ctx.Err() and let the caller
		//    decide. This is not a crash and does not consume budget.
		// 3. Real failure — record and either retry or escalate.
		if err == nil {
			klog.InfoS("Component finished gracefully", "component", s.Name)
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		lastErr = err
		if !s.recordFailure(err) {
			klog.ErrorS(lastErr, "Component restart budget exhausted, escalating", "component", s.Name, "maxFailures", s.Policy.MaxFailures, "window", s.Policy.FailureWindow)
			return fmt.Errorf("%s: %w: %v", s.Name, ErrEscalate, lastErr)
		}
		attempt++
	}
}

// recordFailure stores a new failure timestamp and returns whether the
// supervisor should keep retrying (true) or escalate (false).
//
// Callers must not hold any other locks when invoking this method.
func (s *Supervisor) recordFailure(_ error) bool {
	if s.Policy.MaxFailures <= 0 {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	cutoff := now.Add(-s.Policy.FailureWindow)

	// Drop failures older than the sliding window.
	pruned := s.failures[:0]
	for _, t := range s.failures {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	pruned = append(pruned, now)
	s.failures = pruned

	return len(s.failures) <= s.Policy.MaxFailures
}

// computeBackoff returns an exponentially-growing wait clamped to
// [MinBackoff, MaxBackoff]. attempt is 1-indexed (attempt 1 is the first
// retry, i.e. after the first failure).
func (s *Supervisor) computeBackoff(attempt int) time.Duration {
	if s.Policy.MinBackoff <= 0 {
		return 0
	}
	d := s.Policy.MinBackoff
	for i := 1; i < attempt; i++ {
		d *= 2
		if s.Policy.MaxBackoff > 0 && d >= s.Policy.MaxBackoff {
			return s.Policy.MaxBackoff
		}
	}
	if s.Policy.MaxBackoff > 0 && d > s.Policy.MaxBackoff {
		d = s.Policy.MaxBackoff
	}
	return d
}

func ctxSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
