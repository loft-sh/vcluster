package filters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestParseKubeletContainerLogsPath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantOK        bool
		wantPrefix    string
		wantNamespace string
		wantPod       string
		wantContainer string
	}{
		{
			name:          "exact form",
			path:          "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container",
			wantOK:        true,
			wantPrefix:    "/api/v1/nodes/node-1",
			wantNamespace: "ns",
			wantPod:       "pod",
			wantContainer: "container",
		},
		{
			name:   "trailing traversal to pods",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/../../../../pods",
			wantOK: false,
		},
		{
			name:   "trailing traversal to configz",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/../../../../configz",
			wantOK: false,
		},
		{
			name:   "traversal to second containerLogs",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/../../../other-ns/other-pod/other-container",
			wantOK: false,
		},
		{
			name:   "extra segment without dots",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/extra",
			wantOK: false,
		},
		{
			name:   "trailing slash",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/",
			wantOK: false,
		},
		{
			name:   "missing container",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod",
			wantOK: false,
		},
		{
			name:   "empty segment",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns//container",
			wantOK: false,
		},
		{
			name:   "dot container",
			path:   "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/..",
			wantOK: false,
		},
		{
			name:   "unexpected prefix",
			path:   "/api/v1/nodes/node-1/extra/proxy/containerLogs/ns/pod/container",
			wantOK: false,
		},
		{
			name:   "not a containerLogs path",
			path:   "/api/v1/nodes/node-1/proxy/pods",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, namespace, pod, container, ok := parseKubeletContainerLogsPath(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v for %q, got %v", tt.wantOK, tt.path, ok)
			}
			if !tt.wantOK {
				return
			}
			if prefix != tt.wantPrefix || namespace != tt.wantNamespace || pod != tt.wantPod || container != tt.wantContainer {
				t.Fatalf("expected (%q, %q, %q, %q), got (%q, %q, %q, %q)",
					tt.wantPrefix, tt.wantNamespace, tt.wantPod, tt.wantContainer,
					prefix, namespace, pod, container)
			}
		})
	}
}

func TestEnforceKubeletSubpathAllowlistRejectsUnvalidatedPaths(t *testing.T) {
	s := serializer.NewCodecFactory(scheme.Scheme)
	tests := []struct {
		name       string
		path       string
		sharedNode bool
		wantStatus int
	}{
		{
			name:       "containerLogs with trailing traversal to pods",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/../../../../pods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "containerLogs with trailing traversal on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/../../../../configz",
			sharedNode: false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "traversal ending in an allowed suffix",
			path:       "/api/v1/nodes/node-1/proxy/configz/../healthz",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "current directory segment",
			path:       "/api/v1/nodes/node-1/proxy/./pods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "containerLogs with extra segment",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/extra",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "containerLogs with extra segment on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/extra",
			sharedNode: false,
			wantStatus: http.StatusForbidden,
		},
		// percent-encoded dot segments are decoded into req.URL.Path by the URL parser,
		// so they must be rejected exactly like literal ones
		{
			name:       "encoded traversal ending in an allowed suffix",
			path:       "/api/v1/nodes/node-1/proxy/configz/%2e%2e/healthz",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "uppercase encoded traversal to pods",
			path:       "/api/v1/nodes/node-1/proxy/configz/%2E%2E/pods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "mixed literal and encoded dot segment",
			path:       "/api/v1/nodes/node-1/proxy/configz/.%2e/metrics",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "containerLogs with encoded traversal to pods",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container/%2e%2e/%2e%2e/%2e%2e/%2e%2e/pods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "containerLogs with encoded slashes and dots",
			path:       "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/container%2f%2e%2e%2f%2e%2e%2f%2e%2e%2f%2e%2e%2fpods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		// double encoding decodes to a literal "%2e%2e" segment; it is not a dot segment,
		// but the path no longer exactly matches an allowed subpath
		{
			name:       "double-encoded traversal ending in an allowed suffix",
			path:       "/api/v1/nodes/node-1/proxy/configz/%252e%252e/healthz",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
		// an encoded slash right after /proxy decodes to an empty segment; with host node sync
		// off, unknown subpaths are forwarded, so these must be rejected before classification
		{
			name:       "encoded slash before pods on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/%2Fpods",
			sharedNode: false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "encoded slash before stats summary on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/%2Fstats/summary",
			sharedNode: false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "encoded slash before metrics cadvisor on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/%2Fmetrics/cadvisor",
			sharedNode: false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty segment inside an allowed subpath on dedicated nodes",
			path:       "/api/v1/nodes/node-1/proxy/stats%2F%2Fsummary",
			sharedNode: false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "encoded slash before pods",
			path:       "/api/v1/nodes/node-1/proxy/%2Fpods",
			sharedNode: true,
			wantStatus: http.StatusBadRequest,
		},
		// paths that only end in an allowed subpath must not be classified as that subpath
		{
			name:       "suffix-only healthz under kubelet logs",
			path:       "/api/v1/nodes/node-1/proxy/logs/x/healthz",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "suffix-only pods",
			path:       "/api/v1/nodes/node-1/proxy/configz/pods",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "suffix-only metrics",
			path:       "/api/v1/nodes/node-1/proxy/debug/metrics",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "suffix-only stats",
			path:       "/api/v1/nodes/node-1/proxy/logs/stats/summary",
			sharedNode: true,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the URL parser decodes the path the same way the vcluster's HTTP server does
			req := httptest.NewRequest(http.MethodGet, "http://vcluster.local"+tt.path, nil)
			w := httptest.NewRecorder()

			// a nil register context proves the request is rejected before any lookup or forwarding
			if enforceKubeletSubpathAllowlist(nil, s, w, req, tt.sharedNode) {
				t.Fatalf("expected %q to be rejected", tt.path)
			}
			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d for %q, got %d: %s", tt.wantStatus, tt.path, w.Code, w.Body.String())
			}
		})
	}
}

// With host node sync off the nodes are dedicated to the virtual cluster, so unknown subpaths are
// forwarded by design. Paths that only end in an allowed subpath, or hide one behind double
// encoding, are unknown subpaths: they must be forwarded as sent and must not be classified as an
// allowed subpath, so handleNodeRequest applies no pods, stats or metrics rewrite to them.
func TestEnforceKubeletSubpathAllowlistForwardsUnknownPathsOnDedicatedNodes(t *testing.T) {
	s := serializer.NewCodecFactory(scheme.Scheme)
	tests := []struct {
		name string
		path string
	}{
		{
			name: "suffix-only healthz under kubelet logs",
			path: "/api/v1/nodes/node-1/proxy/logs/x/healthz",
		},
		{
			name: "suffix-only pods",
			path: "/api/v1/nodes/node-1/proxy/configz/pods",
		},
		{
			name: "suffix-only metrics",
			path: "/api/v1/nodes/node-1/proxy/debug/metrics",
		},
		{
			name: "suffix-only stats",
			path: "/api/v1/nodes/node-1/proxy/logs/stats/summary",
		},
		{
			name: "double-encoded traversal ending in an allowed suffix",
			path: "/api/v1/nodes/node-1/proxy/configz/%252e%252e/healthz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://vcluster.local"+tt.path, nil)
			w := httptest.NewRecorder()

			// unknown subpaths are forwarded without a lookup, so a nil register context is enough
			if !enforceKubeletSubpathAllowlist(nil, s, w, req, false) {
				t.Fatalf("expected %q to be forwarded on dedicated nodes, got %d: %s", tt.path, w.Code, w.Body.String())
			}
			if w.Body.Len() != 0 {
				t.Fatalf("expected no response to be written for %q, got: %s", tt.path, w.Body.String())
			}
			if got := req.URL.EscapedPath(); got != tt.path {
				t.Fatalf("expected %q to be forwarded as sent, got %q", tt.path, got)
			}
			if IsKubeletHealthz(req.URL.Path) || IsKubeletStats(req.URL.Path) || IsKubeletMetrics(req.URL.Path) || IsKubeletPods(req.URL.Path) {
				t.Fatalf("expected %q not to be classified as an allowed subpath", req.URL.Path)
			}
		})
	}
}

func TestEnforceKubeletSubpathAllowlistForwardsValidatedPath(t *testing.T) {
	s := serializer.NewCodecFactory(scheme.Scheme)
	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "healthz",
			path:     "/api/v1/nodes/node-1/proxy/healthz",
			wantPath: "/api/v1/nodes/node-1/proxy/healthz",
		},
		{
			name:     "healthz with non-canonical encoding",
			path:     "/api/v1/nodes/node%2D1/proxy/healthz",
			wantPath: "/api/v1/nodes/node-1/proxy/healthz",
		},
		{
			name:     "pods with non-canonical encoding",
			path:     "/api/v1/nodes/node-1/proxy/%70ods",
			wantPath: "/api/v1/nodes/node-1/proxy/pods",
		},
		{
			name:     "stats summary",
			path:     "/api/v1/nodes/node-1/proxy/stats/summary",
			wantPath: "/api/v1/nodes/node-1/proxy/stats/summary",
		},
		{
			name:     "stats summary with non-canonical encoding",
			path:     "/api/v1/nodes/node-1/proxy/stats/%73ummary",
			wantPath: "/api/v1/nodes/node-1/proxy/stats/summary",
		},
		{
			name:     "metrics resource",
			path:     "/api/v1/nodes/node-1/proxy/metrics/resource",
			wantPath: "/api/v1/nodes/node-1/proxy/metrics/resource",
		},
		{
			name:     "metrics resource with non-canonical encoding",
			path:     "/api/v1/nodes/node-1/proxy/metrics/%72esource",
			wantPath: "/api/v1/nodes/node-1/proxy/metrics/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://vcluster.local"+tt.path, nil)
			w := httptest.NewRecorder()

			// healthz, pods, stats and metrics are forwarded without a lookup, so a nil register context is enough
			if !enforceKubeletSubpathAllowlist(nil, s, w, req, true) {
				t.Fatalf("expected %q to be forwarded, got %d: %s", tt.path, w.Code, w.Body.String())
			}
			if req.URL.RawPath != "" {
				t.Fatalf("expected RawPath to be cleared for %q, got %q", tt.path, req.URL.RawPath)
			}
			if got := req.URL.EscapedPath(); got != tt.wantPath {
				t.Fatalf("expected forwarded path %q, got %q", tt.wantPath, got)
			}
		})
	}
}

func TestKubeletPathHelpers(t *testing.T) {
	tests := []struct {
		path        string
		wantHealthz bool
		wantStats   bool
		wantMetrics bool
		wantPods    bool
	}{
		// node proxy form, used by the node proxy filter
		{path: "/api/v1/nodes/node-1/proxy/healthz", wantHealthz: true},
		{path: "/api/v1/nodes/node-1/proxy/stats/summary", wantStats: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics/cadvisor", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics/probes", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics/resource", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics/resource/v1alpha1", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/metrics/resource/v1beta1", wantMetrics: true},
		{path: "/api/v1/nodes/node-1/proxy/pods", wantPods: true},
		{path: "/api/v1/nodes/node-1/proxy/pods/", wantPods: true},
		{path: "/api/v1/nodes/node-1/proxy/runningpods", wantPods: true},
		{path: "/api/v1/nodes/node-1/proxy/logs/x/healthz"},
		{path: "/api/v1/nodes/node-1/proxy/logs/stats/summary"},
		{path: "/api/v1/nodes/node-1/proxy/debug/metrics"},
		{path: "/api/v1/nodes/node-1/proxy/configz/pods"},
		{path: "/api/v1/nodes/node-1/extra/proxy/healthz"},
		{path: "/api/v1/nodes/node-1/proxy"},
		// an empty node name is not a node proxy path; the kubelet authorizer calls these helpers
		// directly, so they must not classify it as an allowed subpath
		{path: "/api/v1/nodes//proxy/healthz"},
		{path: "/api/v1/nodes//proxy/stats/summary"},
		{path: "/api/v1/nodes//proxy/metrics/resource"},
		{path: "/api/v1/nodes//proxy/pods"},
		// kubelet-local form, used by the kubelet authorizer
		{path: "/healthz", wantHealthz: true},
		{path: "/stats/summary", wantStats: true},
		{path: "/metrics/probes", wantMetrics: true},
		{path: "/metrics/resource/v1alpha1", wantMetrics: true},
		{path: "/pods", wantPods: true},
		{path: "/logs/healthz"},
		{path: "/configz/pods"},
		{path: "/logs/stats/summary"},
		{path: "/debug/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsKubeletHealthz(tt.path); got != tt.wantHealthz {
				t.Errorf("IsKubeletHealthz(%q) = %v, want %v", tt.path, got, tt.wantHealthz)
			}
			if got := IsKubeletStats(tt.path); got != tt.wantStats {
				t.Errorf("IsKubeletStats(%q) = %v, want %v", tt.path, got, tt.wantStats)
			}
			if got := IsKubeletMetrics(tt.path); got != tt.wantMetrics {
				t.Errorf("IsKubeletMetrics(%q) = %v, want %v", tt.path, got, tt.wantMetrics)
			}
			if got := IsKubeletPods(tt.path); got != tt.wantPods {
				t.Errorf("IsKubeletPods(%q) = %v, want %v", tt.path, got, tt.wantPods)
			}
		})
	}
}

func TestHasEmptySegment(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/v1/nodes/node-1/proxy/pods", want: false},
		{path: "/api/v1/nodes/node-1/proxy/pods/", want: false},
		{path: "/api/v1/nodes/node-1/proxy//pods", want: true},
		{path: "/api/v1/nodes/node-1/proxy/stats//summary", want: true},
		{path: "/api/v1/nodes/node-1/proxy/pods//", want: true},
	}

	for _, tt := range tests {
		if got := hasEmptySegment(tt.path); got != tt.want {
			t.Fatalf("hasEmptySegment(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestHasDotSegment(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/v1/nodes/node-1/proxy/pods", want: false},
		{path: "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/c..d", want: false},
		{path: "/api/v1/nodes/node-1/proxy/containerLogs/ns/pod/c/..", want: true},
		{path: "/api/v1/nodes/node-1/proxy/./pods", want: true},
	}

	for _, tt := range tests {
		if got := hasDotSegment(tt.path); got != tt.want {
			t.Fatalf("hasDotSegment(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
