//go:build !windows

package osutil

import (
	"sync"
	"testing"
	"time"
)

func TestPreInterruptHandlersFinishBeforeInterruptHandlers(t *testing.T) {
	savedPre, saved := preInterruptHandlers, interruptHandlers
	preInterruptHandlers, interruptHandlers = nil, nil
	t.Cleanup(func() {
		preInterruptHandlers, interruptHandlers = savedPre, saved
		// runInterruptHandlers keeps the lock because the process exits after it.
		interruptExitMu.Unlock()
	})

	var (
		mu    sync.Mutex
		order []string
	)
	record := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, s)
	}

	RegisterInterruptHandler(func() { record("handler") })
	RegisterPreInterruptHandler(func() {
		time.Sleep(50 * time.Millisecond)
		record("pre-1")
	})
	RegisterPreInterruptHandler(func() { record("pre-2") })

	runInterruptHandlers()

	want := []string{"pre-1", "pre-2", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
