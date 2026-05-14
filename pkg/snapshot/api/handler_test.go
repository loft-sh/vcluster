package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

func TestWithSnapshotsDelegatesOtherPaths(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	handler := WithSnapshots(next, &synccontext.ControllerContext{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil))

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}

func TestWithSnapshotsDelegatesSimilarPrefixPaths(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	handler := WithSnapshots(next, &synccontext.ControllerContext{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vcluster/snapshots-extra", nil))

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}

func TestWithSnapshotsProbe(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := WithSnapshots(next, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/vcluster/snapshots", nil))

	if called {
		t.Fatal("expected snapshot handler to handle probe")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", recorder.Body.String())
	}
}

func TestWithSnapshotsRejectsUnknownSnapshotPath(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := WithSnapshots(next, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vcluster/snapshots/unknown/path", nil))

	if called {
		t.Fatal("expected snapshot handler to handle snapshot path")
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestWithSnapshotsRejectsMissingBody(t *testing.T) {
	handler := WithSnapshots(nil, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/vcluster/snapshots/list", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestWithSnapshotsRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "create",
			path: "/vcluster/snapshots",
			body: `{"type":"s3","s3":{"bucket":"bucket"}}`,
		},
		{
			name: "list",
			path: "/vcluster/snapshots/list",
			body: `{"type":"s3"}`,
		},
		{
			name: "create request",
			path: "/vcluster/snapshots/request",
			body: `{"type":"s3","s3":{"bucket":"bucket"}}`,
		},
		{
			name: "delete request",
			path: "/vcluster/snapshots/request/delete",
			body: `{"type":"s3","s3":{"bucket":"bucket"}}`,
		},
	}

	handler := WithSnapshots(nil, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}
		})
	}
}

func TestWithSnapshotsRejectsUnsupportedDeleteRequestMethod(t *testing.T) {
	handler := WithSnapshots(nil, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/vcluster/snapshots/request/delete", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestWithSnapshotsRoutesSnapshotRequestName(t *testing.T) {
	handler := WithSnapshots(nil, &synccontext.ControllerContext{Config: &config.VirtualClusterConfig{}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vcluster/snapshots/request/test-snapshot", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "404") {
		t.Fatalf("expected request name route to match, got body %q", recorder.Body.String())
	}
}

func TestDecodeOptionsAcceptsWrappedAndRawOptions(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "wrapped",
			body: `{"options":{"type":"container","container":{"path":"/tmp/snapshot.tar.gz"}}}`,
		},
		{
			name: "raw",
			body: `{"type":"container","container":{"path":"/tmp/snapshot.tar.gz"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/vcluster/snapshots", strings.NewReader(test.body))
			options, err := snapshotOptions(request, false)
			if err != nil {
				t.Fatalf("decode options: %v", err)
			}
			if options.Type != "container" {
				t.Fatalf("expected type container, got %q", options.Type)
			}
			if options.Container.Path != "/tmp/snapshot.tar.gz" {
				t.Fatalf("expected container path to be decoded, got %q", options.Container.Path)
			}
		})
	}
}

func TestDecodeOptionsRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/vcluster/snapshots", strings.NewReader(strings.Repeat("x", maxRequestBytes+1)))

	_, err := snapshotOptions(request, false)
	if err == nil {
		t.Fatal("expected oversized request body error")
	}
	if !strings.Contains(err.Error(), "request body is too large") {
		t.Fatalf("expected oversized request body error, got %v", err)
	}
}

func TestIsSnapshotPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{path: "/vcluster/snapshots", expected: true},
		{path: "/vcluster/snapshots/list", expected: true},
		{path: "/vcluster/snapshots-extra", expected: false},
		{path: "/api/v1/pods", expected: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := isSnapshotPath(test.path); got != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, got)
			}
		})
	}
}
