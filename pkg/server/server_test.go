package server

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestServiceRedirectAuthResourcesIncludesUpdate(t *testing.T) {
	got := serviceRedirectAuthResources()
	wantVerbs := map[string]bool{"create": false, "update": false}
	svc := corev1.SchemeGroupVersion.WithResource("services")
	for _, resource := range got {
		if resource.GroupVersionResource != svc {
			t.Fatalf("unexpected resource %#v", resource.GroupVersionResource)
		}
		if _, ok := wantVerbs[resource.Verb]; !ok {
			t.Fatalf("unexpected verb %q", resource.Verb)
		}
		wantVerbs[resource.Verb] = true
	}
	for verb, seen := range wantVerbs {
		if !seen {
			t.Fatalf("missing services/%s in redirect authorizer", verb)
		}
	}
}

func TestMetricsAuthNonResources(t *testing.T) {
	wantPaths := map[string]struct{}{
		"/controller-manager/metrics": {},
		"/scheduler/metrics":          {},
		"/metrics/controller-manager": {},
		"/metrics/scheduler":          {},
		"/metrics/etcd":               {},
		"/metrics/kine":               {},
		"/metrics/syncer":             {},
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
