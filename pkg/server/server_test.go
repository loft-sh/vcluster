package server

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestNodeChangesAuthResourcesIncludesUpdateAndPatch(t *testing.T) {
	got := nodeChangesAuthResources()
	nodes := corev1.SchemeGroupVersion.WithResource("nodes")
	want := map[string]bool{
		"update/":       false,
		"update/status": false,
		"patch/":        false,
		"patch/status":  false,
	}
	for _, resource := range got {
		if resource.GroupVersionResource != nodes {
			t.Fatalf("unexpected resource %#v", resource.GroupVersionResource)
		}
		key := resource.Verb + "/" + resource.SubResource
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected nodes auth entry %q", key)
		}
		want[key] = true
	}
	for key, seen := range want {
		if !seen {
			t.Fatalf("missing nodes/%s in redirect authorizer", key)
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
