package server

import "testing"

func TestMetricsAuthNonResources(t *testing.T) {
	wantPaths := map[string]struct{}{
		"/controller-manager/metrics": {},
		"/scheduler/metrics":          {},
		"/metrics/controller-manager": {},
		"/metrics/scheduler":          {},
		"/metrics/etcd":               {},
		"/metrics/kine":               {},
	}

	got := metricsAuthNonResources()
	if len(got) != len(wantPaths) {
		t.Fatalf("expected %d metrics auth paths, got %d", len(wantPaths), len(got))
	}

	for _, pathVerb := range got {
		if _, ok := wantPaths[pathVerb.Path]; !ok {
			t.Fatalf("unexpected metrics auth path %q", pathVerb.Path)
		}
		if pathVerb.Verb != "*" {
			t.Fatalf("expected path %q to delegate all verbs, got %q", pathVerb.Path, pathVerb.Verb)
		}
		delete(wantPaths, pathVerb.Path)
	}
	if len(wantPaths) > 0 {
		t.Fatalf("missing metrics auth paths: %v", wantPaths)
	}
}
