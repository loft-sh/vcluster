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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://vcluster.local"+tt.path, nil)
			req.URL.Path = tt.path
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
