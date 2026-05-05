package command

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock provides a deterministic time source so we don't depend on
// wall-clock advancement for FailureWindow / backoff assertions.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// fakeSleep replaces real time.Sleep in tests; it advances the fake clock by
// the requested duration without actually blocking, but still respects ctx
// cancellation so we can verify cancel-during-backoff behavior.
func fakeSleep(clock *fakeClock, observed *[]time.Duration, observedMu *sync.Mutex) func(ctx context.Context, d time.Duration) error {
	return func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		observedMu.Lock()
		*observed = append(*observed, d)
		observedMu.Unlock()
		clock.Advance(d)
		return nil
	}
}

func newTestSupervisor(t *testing.T, runner ComponentRunner, policy RestartPolicy) (*Supervisor, *fakeClock, *[]time.Duration) {
	t.Helper()
	clock := newFakeClock()
	var observed []time.Duration
	var observedMu sync.Mutex
	return &Supervisor{
		Name:   "test",
		Runner: runner,
		Policy: policy,
		now:    clock.Now,
		sleep:  fakeSleep(clock, &observed, &observedMu),
	}, clock, &observed
}

// TestRestartOnError verifies that a runner that fails a few times then
// succeeds returns nil from the supervisor (graceful completion path).
func TestRestartOnError(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return fmt.Errorf("transient failure %d", n)
		}
		return nil
	}

	s, _, observed := newTestSupervisor(t, runner, RestartPolicy{
		MaxFailures:   5,
		FailureWindow: time.Minute,
		MinBackoff:    time.Second,
		MaxBackoff:    10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Run(ctx); err != nil {
		t.Fatalf("expected nil after graceful completion, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected 3 runner invocations, got %d", got)
	}
	if len(*observed) != 2 {
		t.Errorf("expected 2 backoff sleeps before the third attempt, got %d (%v)", len(*observed), *observed)
	}
}

// TestEscalateAfterBudget verifies that exceeding MaxFailures within
// FailureWindow returns an ErrEscalate-wrapped error.
func TestEscalateAfterBudget(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("always fails")
	}

	s, _, _ := newTestSupervisor(t, runner, RestartPolicy{
		MaxFailures:   3,
		FailureWindow: time.Minute,
		MinBackoff:    time.Millisecond,
		MaxBackoff:    time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Run(ctx)
	if !errors.Is(err, ErrEscalate) {
		t.Fatalf("expected ErrEscalate, got %v", err)
	}
	// MaxFailures=3 means the supervisor tolerates 3 failures and escalates
	// on the 4th.
	if got := atomic.LoadInt32(&attempts); got != 4 {
		t.Errorf("expected 4 runner invocations (3 retries + escalating attempt), got %d", got)
	}
}

// TestExponentialBackoff verifies the backoff schedule grows geometrically and
// caps at MaxBackoff.
func TestExponentialBackoff(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("fail")
	}

	policy := RestartPolicy{
		MaxFailures:   100,
		FailureWindow: time.Hour,
		MinBackoff:    time.Second,
		MaxBackoff:    8 * time.Second,
	}
	s, _, observed := newTestSupervisor(t, runner, policy)

	// Cancel after a few attempts so the test terminates.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Wait until at least 5 attempts have been observed, then cancel.
		for atomic.LoadInt32(&attempts) < 5 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	if err := s.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	want := []time.Duration{
		1 * time.Second, // attempt 1: MinBackoff
		2 * time.Second, // attempt 2: doubled
		4 * time.Second, // attempt 3: doubled
		8 * time.Second, // attempt 4: MaxBackoff (cap)
	}
	if len(*observed) < len(want) {
		t.Fatalf("expected at least %d observed sleeps, got %d (%v)", len(want), len(*observed), *observed)
	}
	for i, d := range want {
		if (*observed)[i] != d {
			t.Errorf("backoff[%d] = %v, want %v", i, (*observed)[i], d)
		}
	}
	// All subsequent observed sleeps should also be capped at MaxBackoff.
	for i := len(want); i < len(*observed); i++ {
		if (*observed)[i] != policy.MaxBackoff {
			t.Errorf("backoff[%d] = %v, want capped at %v", i, (*observed)[i], policy.MaxBackoff)
		}
	}
}

// TestContextCancelDuringRun verifies the supervisor returns ctx.Err()
// without retrying when ctx is cancelled while the runner is executing.
func TestContextCancelDuringRun(t *testing.T) {
	runnerStarted := make(chan struct{})
	runner := func(ctx context.Context) error {
		close(runnerStarted)
		<-ctx.Done()
		return ctx.Err()
	}

	s, _, _ := newTestSupervisor(t, runner, DefaultRestartPolicy())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-runnerStarted
		cancel()
	}()

	err := s.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// TestContextCancelDuringBackoff verifies that ctx cancellation interrupts a
// pending backoff sleep.
func TestContextCancelDuringBackoff(t *testing.T) {
	// Use real time here (briefly) so cancellation actually races with the
	// sleep loop; the fake sleep checks ctx and returns immediately.
	runner := func(_ context.Context) error {
		return errors.New("fail once")
	}
	s := &Supervisor{
		Name:   "test",
		Runner: runner,
		Policy: RestartPolicy{
			MaxFailures:   10,
			FailureWindow: time.Minute,
			MinBackoff:    50 * time.Millisecond,
			MaxBackoff:    50 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel while supervisor is sleeping during backoff.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := s.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// TestGracefulCompletionDoesNotRestart verifies that a runner returning nil
// causes the supervisor to return nil immediately, without invoking the runner
// again.
func TestGracefulCompletionDoesNotRestart(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return nil
	}
	s, _, _ := newTestSupervisor(t, runner, DefaultRestartPolicy())

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("expected exactly 1 runner invocation, got %d", got)
	}
}

// TestFailureWindowReset verifies that failures spread across more than
// FailureWindow do not accumulate, so a long-running but infrequently-failing
// component is supported indefinitely.
func TestFailureWindowReset(t *testing.T) {
	clock := newFakeClock()
	var attempts int32
	maxAttempts := int32(10)

	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n >= maxAttempts {
			return nil
		}
		return errors.New("periodic failure")
	}

	s := &Supervisor{
		Name:   "test",
		Runner: runner,
		Policy: RestartPolicy{
			MaxFailures:   2,
			FailureWindow: time.Second,
			MinBackoff:    time.Millisecond,
			MaxBackoff:    time.Millisecond,
		},
		now: clock.Now,
		// Custom sleep that also advances the clock past the failure
		// window so each failure looks isolated.
		sleep: func(ctx context.Context, d time.Duration) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			clock.Advance(2 * time.Second)
			return nil
		},
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected supervisor to keep restarting and finish gracefully, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, got)
	}
}

// TestMaxFailuresZeroEscalatesImmediately verifies the documented
// "MaxFailures==0 disables restart" behavior.
func TestMaxFailuresZeroEscalatesImmediately(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("fail")
	}
	s, _, _ := newTestSupervisor(t, runner, RestartPolicy{
		MaxFailures: 0,
	})

	err := s.Run(context.Background())
	if !errors.Is(err, ErrEscalate) {
		t.Fatalf("expected ErrEscalate, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("expected exactly 1 runner invocation, got %d", got)
	}
}

// TestMetricsCallbacks verifies the metrics interface receives the expected
// number of restart and exit events.
func TestMetricsCallbacks(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return fmt.Errorf("fail %d", n)
		}
		return nil
	}

	m := &recordingMetrics{}
	s, _, _ := newTestSupervisor(t, runner, RestartPolicy{
		MaxFailures:   5,
		FailureWindow: time.Minute,
		MinBackoff:    time.Millisecond,
		MaxBackoff:    time.Millisecond,
	})
	s.Metrics = m

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got, want := m.restartCount(), 2; got != want {
		t.Errorf("ObserveRestart count = %d, want %d", got, want)
	}
	if got, want := m.exitCount(), 3; got != want {
		t.Errorf("ObserveExit count = %d, want %d", got, want)
	}
}

type recordingMetrics struct {
	mu       sync.Mutex
	restarts []restartEvent
	exits    []exitEvent
}

type restartEvent struct {
	component string
	attempt   int
	backoff   time.Duration
}

type exitEvent struct {
	component string
	err       error
}

func (m *recordingMetrics) ObserveRestart(component string, attempt int, backoff time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restarts = append(m.restarts, restartEvent{component, attempt, backoff})
}

func (m *recordingMetrics) ObserveExit(component string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exits = append(m.exits, exitEvent{component, err})
}

func (m *recordingMetrics) restartCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.restarts)
}

func (m *recordingMetrics) exitCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.exits)
}

// TestRunWithRestartFunc smoke-tests the convenience function wrapper.
func TestRunWithRestartFunc(t *testing.T) {
	var attempts int32
	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			return errors.New("once")
		}
		return nil
	}
	policy := RestartPolicy{
		MaxFailures:   3,
		FailureWindow: time.Second,
		MinBackoff:    time.Microsecond,
		MaxBackoff:    time.Microsecond,
	}
	if err := RunWithRestart(context.Background(), "test", runner, policy, nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("expected 2 runner invocations, got %d", got)
	}
}
