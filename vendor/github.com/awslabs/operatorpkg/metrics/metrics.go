package metrics

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
	clientmetrics "k8s.io/client-go/tools/metrics"
)

// This package adds client-go metrics that can be surfaced through the Prometheus metrics server
// This is based on the reference implementation that was pulled out in controller-runtime in https://github.com/kubernetes-sigs/controller-runtime/pull/2298

// RegisterClientMetrics sets up the client latency and result metrics from client-go.
func RegisterClientMetrics(r prometheus.Registerer) {
	clientmetrics.RequestLatency = &LatencyAdapter{Metric: NewPrometheusHistogram(
		r,
		prometheus.HistogramOpts{
			Name:    "client_go_request_duration_seconds",
			Help:    "Request latency in seconds. Broken down by verb, group, version, kind, and subresource.",
			Buckets: prometheus.ExponentialBuckets(0.001, 1.5, 20),
		},
		[]string{"verb", "group", "version", "kind", "subresource"},
	)}
	clientmetrics.RequestResult = &ResultAdapter{Metric: NewPrometheusCounter(
		r,
		prometheus.CounterOpts{
			Name: "client_go_request_total",
			Help: "Number of HTTP requests, partitioned by status code and method.",
		},
		[]string{"code", "method"},
	)}
}

type ResultAdapter struct {
	Metric CounterMetric
}

func (r *ResultAdapter) Increment(_ context.Context, code, method, _ string) {
	r.Metric.Inc(map[string]string{"code": code, "method": method})
}

// LatencyAdapter implements LatencyMetric.
type LatencyAdapter struct {
	Metric ObservationMetric
}

// Observe increments the request latency metric for the given verb/group/version/kind/subresource.
func (l *LatencyAdapter) Observe(_ context.Context, verb string, u url.URL, latency time.Duration) {
	if data := parsePath(u.Path); data != nil {
		// We update the "verb" to better reflect the action being taken by client-go
		switch verb {
		case "POST":
			verb = "CREATE"
		case "GET":
			if !strings.Contains(u.Path, "{name}") {
				verb = "LIST"
			}
		case "PUT":
			if !strings.Contains(u.Path, "{name}") {
				verb = "CREATE"
			} else {
				verb = "UPDATE"
			}
		}
		l.Metric.Observe(latency.Seconds(), map[string]string{
			"verb":        verb,
			"group":       data.group,
			"version":     data.version,
			"kind":        data.kind,
			"subresource": data.subresource,
		})
	}
}

// pathData stores data parsed out from the URL path
type pathData struct {
	group       string
	version     string
	kind        string
	subresource string
}

// parsePath parses out the URL called from client-go to return back the group, version, kind, and subresource
// urls are formatted similar to /apis/coordination.k8s.io/v1/namespaces/{namespace}/leases/{name} or /apis/karpenter.sh/v1beta1/nodeclaims/{name}
func parsePath(path string) *pathData {
	parts := strings.Split(path, "/")[1:]

	var groupIdx, versionIdx, kindIdx int
	switch parts[0] {
	case "api":
		groupIdx = 0
	case "apis":
		groupIdx = 1
	default:
		return nil
	}
	// If the url is too short, then it's not interesting to us
	if len(parts) < groupIdx+3 {
		return nil
	}
	// This resource is namespaced and the resource is not the namespace
	if parts[groupIdx+2] == "namespaces" && len(parts) > groupIdx+4 {
		versionIdx = groupIdx + 1
		kindIdx = versionIdx + 3
	} else {
		versionIdx = groupIdx + 1
		kindIdx = versionIdx + 1
	}

	// If we have a subresource, it's going to be two indices after the kind
	var subresource string
	if len(parts) == kindIdx+3 {
		subresource = parts[kindIdx+2]
	}
	return &pathData{
		// If the group index is 0, this is part of the core API, so there's no group
		group:       lo.Ternary(groupIdx == 0, "", parts[groupIdx]),
		version:     parts[versionIdx],
		kind:        parts[kindIdx],
		subresource: subresource,
	}
}
