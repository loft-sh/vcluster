// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows

package osutil

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"k8s.io/klog/v2"
)

// InterruptHandler is a function that is called on receiving a
// SIGTERM or SIGINT signal.
type InterruptHandler func()

var (
	interruptRegisterMu, interruptExitMu sync.Mutex
	// interruptHandlers holds all registered InterruptHandlers in order
	// they will be executed.
	interruptHandlers []InterruptHandler
)

// RegisterInterruptHandler registers a new InterruptHandler. Handlers registered
// after interrupt handing was initiated will not be executed.
func RegisterInterruptHandler(h InterruptHandler) {
	interruptRegisterMu.Lock()
	defer interruptRegisterMu.Unlock()
	interruptHandlers = append(interruptHandlers, h)
}

// HandleInterrupts calls the handler functions on receiving a SIGINT or SIGTERM.
func HandleInterrupts() {
	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-notifier
		klog.FromContext(context.Background()).Info("received signal; shutting down", "signal", sig.String())

		runInterruptHandlers()
		signal.Stop(notifier)
		os.Exit(0)
	}()
}

// Exit relays to os.Exit if no interrupt handlers are running, blocks otherwise.
func Exit(code int) {
	runInterruptHandlers()
	os.Exit(code)
}

func runInterruptHandlers() {
	klog.FromContext(context.Background()).Info("running interrupt handlers")

	interruptRegisterMu.Lock()
	ihs := make([]InterruptHandler, len(interruptHandlers))
	copy(ihs, interruptHandlers)
	interruptRegisterMu.Unlock()

	interruptExitMu.Lock()
	waitGroup := sync.WaitGroup{}
	for _, h := range ihs {
		waitGroup.Add(1)
		go func(h InterruptHandler) {
			defer waitGroup.Done()
			h()
		}(h)
	}
	waitGroup.Wait()
}
