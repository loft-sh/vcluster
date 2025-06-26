<<<<<<<< HEAD:vendor/github.com/prometheus/client_golang/prometheus/promhttp/internal/compression.go
// Copyright 2025 The Prometheus Authors
========
// Copyright 2023 The Prometheus Authors
>>>>>>>> main:vendor/github.com/prometheus/client_golang/prometheus/process_collector_not_supported.go
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

<<<<<<<< HEAD:vendor/github.com/prometheus/client_golang/prometheus/promhttp/internal/compression.go
package internal
========
//go:build wasip1 || js || ios
// +build wasip1 js ios
>>>>>>>> main:vendor/github.com/prometheus/client_golang/prometheus/process_collector_not_supported.go

import (
	"io"
)

<<<<<<<< HEAD:vendor/github.com/prometheus/client_golang/prometheus/promhttp/internal/compression.go
// NewZstdWriter enables zstd write support if non-nil.
var NewZstdWriter func(rw io.Writer) (_ io.Writer, closeWriter func(), _ error)
========
func canCollectProcess() bool {
	return false
}

func (c *processCollector) processCollect(ch chan<- Metric) {
	c.errorCollectFn(ch)
}

// describe returns all descriptions of the collector for wasip1 and js.
// Ensure that this list of descriptors is kept in sync with the metrics collected
// in the processCollect method. Any changes to the metrics in processCollect
// (such as adding or removing metrics) should be reflected in this list of descriptors.
func (c *processCollector) describe(ch chan<- *Desc) {
	c.errorDescribeFn(ch)
}
>>>>>>>> main:vendor/github.com/prometheus/client_golang/prometheus/process_collector_not_supported.go
