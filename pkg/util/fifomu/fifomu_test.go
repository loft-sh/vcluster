// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package fifomu_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/fifomu"
)

// Acknowledgement: Much of the test code in this file is
// copied from stdlib sync/mutex_test.go.

// mutexer is the exported methodset of sync.Mutex.
type mutexer interface {
	sync.Locker
	TryLock() bool
}

var (
	_ mutexer = (*fifomu.Mutex)(nil)
	_ mutexer = (*sync.Mutex)(nil)
)

// newMu is a function that returns a new mutexer.
// We set it to newFifoMu, newStdlibMu or newSemaphoreMu
// for benchmarking.
var newMu = newFifoMu

func newFifoMu() mutexer {
	return &fifomu.Mutex{}
}

func HammerMutex(m mutexer, loops int, cdone chan bool) {
	for i := 0; i < loops; i++ {
		if i%3 == 0 {
			if m.TryLock() {
				m.Unlock()
			}
			continue
		}
		m.Lock()
		m.Unlock() //nolint:staticcheck
	}
	cdone <- true
}

func TestMutex(t *testing.T) {
	if n := runtime.SetMutexProfileFraction(1); n != 0 {
		t.Logf("got mutexrate %d expected 0", n)
	}
	defer runtime.SetMutexProfileFraction(0)

	m := newMu()

	m.Lock()
	if m.TryLock() {
		t.Fatalf("TryLock succeeded with mutex locked")
	}
	m.Unlock()
	if !m.TryLock() {
		t.Fatalf("TryLock failed with mutex unlocked")
	}
	m.Unlock()

	c := make(chan bool)
	for i := 0; i < 10; i++ {
		go HammerMutex(m, 1000, c)
	}
	for i := 0; i < 10; i++ {
		<-c
	}
}

func TestMutexMisuse(t *testing.T) {
	t.Run("Mutex.Unlock", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic due to Unlock of unlocked mutex")
			}
		}()

		mu := newMu()
		mu.Unlock()
	})

	t.Run("Mutex.Unlock2", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic due to Unlock of unlocked mutex")
			}
		}()

		mu := newMu()
		mu.Lock()
		mu.Unlock() //nolint:staticcheck
		mu.Unlock()
	})
}

func TestMutexFairness(t *testing.T) {
	mu := newMu()
	stop := make(chan bool)
	defer close(stop)
	go func() {
		for {
			mu.Lock()
			time.Sleep(100 * time.Microsecond)
			mu.Unlock()
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
	done := make(chan bool, 1)
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Microsecond)
			mu.Lock()
			mu.Unlock() //nolint:staticcheck
		}
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("can't acquire mutex in 10 seconds")
	}
}
