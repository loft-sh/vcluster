package cli

import (
	"context"
	"fmt"
	"testing"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestWaitUntilNotFoundSucceedsOnNotFound(t *testing.T) {
	err := waitUntilNotFound(context.Background(), time.Millisecond, time.Second, func(context.Context) error {
		return kerrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, "gone")
	})
	if err != nil {
		t.Fatalf("expected success when object is already gone, got %v", err)
	}
}

func TestWaitUntilNotFoundRejectsForbidden(t *testing.T) {
	forbidden := kerrors.NewForbidden(schema.GroupResource{Resource: "namespaces"}, "ns", fmt.Errorf("denied"))
	err := waitUntilNotFound(context.Background(), time.Millisecond, time.Second, func(context.Context) error {
		return forbidden
	})
	if !kerrors.IsForbidden(err) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestWaitUntilNotFoundReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := waitUntilNotFound(ctx, time.Millisecond, time.Second, func(context.Context) error {
		return ctx.Err()
	})
	if err == nil {
		t.Fatal("expected context error, treated cancel as successful delete")
	}
}

func TestWaitUntilNotFoundTimesOutWhilePresent(t *testing.T) {
	err := waitUntilNotFound(context.Background(), time.Millisecond, 20*time.Millisecond, func(context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected timeout while object still exists")
	}
}
