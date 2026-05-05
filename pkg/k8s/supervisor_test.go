package k8s

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/command"
)

// fastPolicy is a low-backoff policy used by integration tests in this
// package so we don't sleep for seconds between attempts.
var fastPolicy = command.RestartPolicy{
	MaxFailures:   5,
	FailureWindow: time.Minute,
	MinBackoff:    time.Microsecond,
	MaxBackoff:    time.Microsecond,
}

// withFastPolicy installs fastPolicy for the duration of the test.
func withFastPolicy(t *testing.T) {
	t.Helper()
	prev := supervisorPolicy
	t.Cleanup(func() { supervisorPolicy = prev })
	supervisorPolicy = fastPolicy
}

// TestRunComponentSupervisorEnabled exercises the integration shape used by
// each goroutine in pkg/k8s/k8s.go: build a runner closure, call runComponent,
// expect graceful completion after a couple of transient failures.
func TestRunComponentSupervisorEnabled(t *testing.T) {
	t.Setenv(ComponentSupervisorEnvVar, "true")
	withFastPolicy(t)

	var attempts int32
	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errors.New("transient")
		}
		return nil
	}

	if err := runComponent(context.Background(), "test-component", runner); err != nil {
		t.Fatalf("expected nil after graceful completion, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected 3 runner invocations, got %d", got)
	}
}

// TestRunComponentSupervisorDisabled checks that with the env var explicitly
// off, the legacy single-shot path runs and any error propagates immediately.
func TestRunComponentSupervisorDisabled(t *testing.T) {
	t.Setenv(ComponentSupervisorEnvVar, "false")

	var attempts int32
	wantErr := errors.New("legacy fail")
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return wantErr
	}

	err := runComponent(context.Background(), "test-component", runner)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("expected exactly 1 runner invocation in legacy mode, got %d", got)
	}
}

// TestRunComponentEscalation verifies that exhausting the budget propagates
// ErrEscalate so the goroutine in k8s.go can map it to osutil.Exit(1).
//
// We override supervisorPolicy locally with sub-millisecond backoffs so the
// test runs in milliseconds rather than the 30+ s the real default would cost
// after exponential backoff.
func TestRunComponentEscalation(t *testing.T) {
	t.Setenv(ComponentSupervisorEnvVar, "true")

	prev := supervisorPolicy
	t.Cleanup(func() { supervisorPolicy = prev })
	supervisorPolicy = command.RestartPolicy{
		MaxFailures:   2, // smaller budget than fastPolicy to keep iterations bounded
		FailureWindow: time.Minute,
		MinBackoff:    time.Microsecond,
		MaxBackoff:    time.Microsecond,
	}

	runner := func(_ context.Context) error { return errors.New("always") }

	err := runComponent(context.Background(), "test-component", runner)
	if !errors.Is(err, command.ErrEscalate) {
		t.Fatalf("expected ErrEscalate, got %v", err)
	}
}

// TestRunComponentInvalidEnvFallsBackToLegacy guards the parsing branch: an
// unrecognized env value should not silently enable the supervisor — we want
// the safer path.
func TestRunComponentInvalidEnvFallsBackToLegacy(t *testing.T) {
	t.Setenv(ComponentSupervisorEnvVar, "yes-please")

	var attempts int32
	wantErr := errors.New("legacy fail")
	runner := func(_ context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return wantErr
	}

	err := runComponent(context.Background(), "test-component", runner)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wantErr, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("expected exactly 1 runner invocation when env value is invalid, got %d", got)
	}
}

// TestRunComponentEnvDefault verifies the local-validation default (env unset
// → supervisor enabled). Restores the env so subsequent tests don't see this
// override.
func TestRunComponentEnvDefault(t *testing.T) {
	// t.Setenv with empty string still sets the var; use Unsetenv so we
	// hit the actual "not set" branch in componentSupervisorEnabled.
	prev, hadPrev := lookupEnv(ComponentSupervisorEnvVar)
	if hadPrev {
		t.Cleanup(func() { _ = setEnv(ComponentSupervisorEnvVar, prev) })
	} else {
		t.Cleanup(func() { _ = unsetEnv(ComponentSupervisorEnvVar) })
	}
	if err := unsetEnv(ComponentSupervisorEnvVar); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	withFastPolicy(t)

	var attempts int32
	runner := func(_ context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			return errors.New("once")
		}
		return nil
	}

	if err := runComponent(context.Background(), "test-component", runner); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("expected supervisor to retry once and then succeed, got %d attempts", got)
	}
}
