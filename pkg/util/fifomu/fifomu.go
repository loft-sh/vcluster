// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fifomu provides a Mutex whose Lock method returns the lock to
// callers in FIFO call order. This is in contrast to sync.Mutex, where
// a single goroutine can repeatedly lock and unlock and relock the mutex
// without handing off to other lock waiter goroutines (that is, until after
// a 1ms starvation threshold, at which point sync.Mutex enters a FIFO
// "starvation mode" for those starved waiters, but that's too late for some
// use cases).
//
// fifomu.Mutex implements the exported methods of sync.Mutex and thus is
// a drop-in replacement (and by extension also implements sync.Locker).
// It also provides a bonus context-aware Mutex.LockContext method.
//
// Note: unless you need the FIFO behavior, you should prefer sync.Mutex.
// For typical workloads, its "greedy-relock" behavior requires less goroutine
// switching and yields better performance.
package fifomu

import (
	"context"
	"sync"
)

var _ sync.Locker = (*Mutex)(nil)

// Mutex is a mutual exclusion lock whose Lock method returns
// the lock to callers in FIFO call order.
//
// A Mutex must not be copied after first use.
//
// The zero value for a Mutex is an unlocked mutex.
//
// Mutex implements the same methodset as sync.Mutex, so it can
// be used as a drop-in replacement. It implements an additional
// method Mutex.LockContext, which provides context-aware locking.
type Mutex struct {
	waiters list[waiter]
	cur     int64
	mu      sync.Mutex
}

// Lock locks m.
//
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *Mutex) Lock() {
	m.mu.Lock()
	if m.cur <= 0 && m.waiters.len == 0 {
		m.cur++
		m.mu.Unlock()
		return
	}

	w := waiterPool.Get().(waiter) //nolint:errcheck
	m.waiters.pushBack(w)
	m.mu.Unlock()

	<-w
	waiterPool.Put(w)
}

// LockContext locks m.
//
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available or ctx is done.
//
// On failure, LockContext returns context.Cause(ctx) and
// leaves the mutex unchanged.
//
// If ctx is already done, LockContext may still succeed without blocking.
func (m *Mutex) LockContext(ctx context.Context) error {
	m.mu.Lock()
	if m.cur <= 0 && m.waiters.len == 0 {
		m.cur++
		m.mu.Unlock()
		return nil
	}

	w := waiterPool.Get().(waiter) //nolint:errcheck
	elem := m.waiters.pushBackElem(w)
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		err := context.Cause(ctx)
		m.mu.Lock()
		select {
		case <-w:
			// Acquired the lock after we were canceled.  Rather than trying to
			// fix up the queue, just pretend we didn't notice the cancellation.
			err = nil
			waiterPool.Put(w)
		default:
			isFront := m.waiters.front() == elem
			m.waiters.remove(elem)
			// If we're at the front and there's extra tokens left,
			// notify other waiters.
			if isFront && m.cur < 1 {
				m.notifyWaiters()
			}
		}
		m.mu.Unlock()
		return err

	case <-w:
		waiterPool.Put(w)
		return nil
	}
}

// TryLock tries to lock m and reports whether it succeeded.
func (m *Mutex) TryLock() bool {
	m.mu.Lock()
	success := m.cur <= 0 && m.waiters.len == 0
	if success {
		m.cur++
	}
	m.mu.Unlock()
	return success
}

// Unlock unlocks m.
// It is a run-time error if m is not locked on entry to Unlock.
//
// A locked Mutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a Mutex and then
// arrange for another goroutine to unlock it.
func (m *Mutex) Unlock() {
	m.mu.Lock()
	m.cur--
	if m.cur < 0 {
		m.mu.Unlock()
		panic("sync: unlock of unlocked mutex")
	}
	m.notifyWaiters()
	m.mu.Unlock()
}

func (m *Mutex) notifyWaiters() {
	for {
		next := m.waiters.front()
		if next == nil {
			break // No more waiters blocked.
		}

		w := next.Value
		if m.cur > 0 {
			// Anti-starvation measure: we could keep going, but under load
			// that could cause starvation for large requests; instead, we leave
			// all remaining waiters blocked.
			break
		}

		m.cur++
		m.waiters.remove(next)
		w <- struct{}{}
	}
}

var waiterPool = sync.Pool{New: func() any { return waiter(make(chan struct{})) }}

type waiter chan struct{}
