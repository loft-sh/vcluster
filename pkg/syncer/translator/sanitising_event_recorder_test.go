package translator

import (
	"fmt"
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// captureRecorder records the sanitised message and pass-through fields from
// the last Eventf call.
type captureRecorder struct {
	lastMessage string
	eventtype   string
	reason      string
	action      string
}

func (c *captureRecorder) Eventf(_ runtime.Object, _ runtime.Object, eventtype, reason, action, note string, args ...any) {
	c.lastMessage = fmt.Sprintf(note, args...)
	c.eventtype = eventtype
	c.reason = reason
	c.action = action
}

// namespacedMapper translates using the standard SingleNamespaceHostName scheme.
type namespacedMapper struct{}

func (m *namespacedMapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return translate.Default.HostName(nil, req.Name, req.Namespace)
}

func (m *namespacedMapper) HostToVirtual(_ *synccontext.SyncContext, _ types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{}
}

func (m *namespacedMapper) IsManaged(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}

func (m *namespacedMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (m *namespacedMapper) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{}
}

// clusterScopedMapper translates using the HostNameCluster scheme (no namespace).
type clusterScopedMapper struct{}

func (m *clusterScopedMapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translate.Default.HostNameCluster(req.Name)}
}

func (m *clusterScopedMapper) HostToVirtual(_ *synccontext.SyncContext, _ types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{}
}

func (m *clusterScopedMapper) IsManaged(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}

func (m *clusterScopedMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (m *clusterScopedMapper) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{}
}

func TestSanitisingEventRecorder(t *testing.T) {
	const vcName = "my-vc"

	origVClusterName := translate.VClusterName
	translate.VClusterName = vcName
	t.Cleanup(func() { translate.VClusterName = origVClusterName })

	pod := func(name, namespace string) *corev1.Pod {
		return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	}

	tests := []struct {
		name        string
		mapper      synccontext.Mapper
		regarding   runtime.Object
		note        string
		args        []any
		wantMessage string
	}{
		{
			name:        "plain host name replaced with virtual name",
			mapper:      &namespacedMapper{},
			regarding:   pod("cuda-vector-add", "default"),
			note:        `Error syncing: update object: Operation cannot be fulfilled on pods "` + translate.SingleNamespaceHostName("cuda-vector-add", "default", vcName) + `": the object has been modified`,
			wantMessage: `Error syncing: update object: Operation cannot be fulfilled on pods "cuda-vector-add": the object has been modified`,
		},
		{
			// Names that exceed 63 chars are hashed by SafeConcatName; Pass 1 must
			// still replace the hash with the virtual name via the exact-match lookup.
			name:        "hashed host name replaced with virtual name",
			mapper:      &namespacedMapper{},
			regarding:   pod("very-long-pod-name-that-definitely-exceeds-63-chars", "default"),
			note:        `Error syncing: pods "` + translate.SingleNamespaceHostName("very-long-pod-name-that-definitely-exceeds-63-chars", "default", vcName) + `" failed`,
			wantMessage: `Error syncing: pods "very-long-pod-name-that-definitely-exceeds-63-chars" failed`,
		},
		{
			name:        "host name not present in message passes through unchanged",
			mapper:      &namespacedMapper{},
			regarding:   pod("my-pod", "default"),
			note:        "Error syncing: some unrelated error occurred",
			wantMessage: "Error syncing: some unrelated error occurred",
		},
		{
			name:        "multiple occurrences of host name all replaced",
			mapper:      &namespacedMapper{},
			regarding:   pod("my-pod", "ns1"),
			note:        "object " + translate.SingleNamespaceHostName("my-pod", "ns1", vcName) + " conflicts with " + translate.SingleNamespaceHostName("my-pod", "ns1", vcName),
			wantMessage: "object my-pod conflicts with my-pod",
		},
		{
			name:        "format args are expanded before sanitisation",
			mapper:      &namespacedMapper{},
			regarding:   pod("web", "production"),
			note:        "Error syncing: %v",
			args:        []any{`update object: Operation cannot be fulfilled on pods "` + translate.SingleNamespaceHostName("web", "production", vcName) + `"`},
			wantMessage: `Error syncing: update object: Operation cannot be fulfilled on pods "web"`,
		},
		{
			// Pod with an empty namespace: SingleNamespaceHostName embeds an empty
			// namespace segment ("name-x--x-vcName"). Pass 1 replaces via exact match;
			// Pass 2 does not fire because namespace is empty.
			name:        "empty namespace host name replaced",
			mapper:      &namespacedMapper{},
			regarding:   pod("my-pod", ""),
			note:        `Error syncing: pods "` + translate.SingleNamespaceHostName("my-pod", "", vcName) + `" failed`,
			wantMessage: `Error syncing: pods "my-pod" failed`,
		},
		{
			name:        "non-pod object (PVC) host name replaced",
			mapper:      &namespacedMapper{},
			regarding:   &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "team-a"}},
			note:        `Error syncing: Operation cannot be fulfilled on persistentvolumeclaims "` + translate.SingleNamespaceHostName("my-pvc", "team-a", vcName) + `"`,
			wantMessage: `Error syncing: Operation cannot be fulfilled on persistentvolumeclaims "my-pvc"`,
		},
		{
			name:        "nil regarding does not panic and passes message through",
			regarding:   nil,
			note:        "Error syncing: %v",
			args:        []any{"some error"},
			wantMessage: "Error syncing: some error",
		},
		{
			// When the API server rejects a pod update because a secondary resource
			// (e.g. a translated ConfigMap) is missing, the error message embeds the
			// host-side name of that secondary resource — not the pod's host name.
			// The suffix-stripping pass must strip it even though it is not the regarding object.
			name:      "secondary resource host name replaced",
			mapper:    &namespacedMapper{},
			regarding: pod("my-pod", "default"),
			note:      `Error syncing: %v`,
			args: []any{
				`configmap "` + translate.SingleNamespaceHostName("app-config", "default", vcName) + `" not found`,
			},
			wantMessage: `Error syncing: configmap "app-config" not found`,
		},
		{
			// A message that embeds both the regarding pod's translated name and a
			// secondary configmap's translated name — both must be replaced.
			name:      "both regarding and secondary host names replaced in same message",
			mapper:    &namespacedMapper{},
			regarding: pod("my-pod", "default"),
			note: `Error syncing: update object: Operation cannot be fulfilled on pods "` +
				translate.SingleNamespaceHostName("my-pod", "default", vcName) +
				`": conflict; referenced configmap "` +
				translate.SingleNamespaceHostName("app-config", "default", vcName) +
				`" not found`,
			wantMessage: `Error syncing: update object: Operation cannot be fulfilled on pods "my-pod": conflict; referenced configmap "app-config" not found`,
		},
		{
			// A resource name that itself contains "-x-" (valid in Kubernetes).
			// Pass 2 splits on vName so "my-x-pod" is recovered correctly.
			name:        "resource name containing -x- replaced correctly",
			mapper:      &namespacedMapper{},
			regarding:   pod("my-x-pod", "default"),
			note:        `Error syncing: pods "` + translate.SingleNamespaceHostName("my-x-pod", "default", vcName) + `" conflict`,
			wantMessage: `Error syncing: pods "my-x-pod" conflict`,
		},
		{
			// When the namespace itself contains "-x-", the exact-suffix approach matches
			// "-x-team-x-blue-x-my-vc" literally and strips it in one operation.
			name:      "secondary resource in namespace containing -x- is replaced correctly",
			mapper:    &namespacedMapper{},
			regarding: pod("my-pod", "team-x-blue"),
			note:      `Error syncing: %v`,
			args: []any{
				`configmap "` + translate.SingleNamespaceHostName("app-config", "team-x-blue", vcName) + `" not found`,
			},
			wantMessage: `Error syncing: configmap "app-config" not found`,
		},
		{
			// A virtual name that ends with the translated suffix pattern is valid in
			// Kubernetes. Pass 2 detects this via HasSuffix and is skipped entirely to
			// avoid corrupting the name that Pass 1 just placed in the message.
			name:        "regarding name ending with suffix is not corrupted by Pass 2",
			mapper:      &namespacedMapper{},
			regarding:   pod("api-x-default-x-my-vc", "default"),
			note:        `Error syncing: pods "` + translate.SingleNamespaceHostName("api-x-default-x-my-vc", "default", vcName) + `" conflict`,
			wantMessage: `Error syncing: pods "api-x-default-x-my-vc" conflict`,
		},
		{
			// The vClusterName suffix appears in the message but there is no preceding
			// "-x-" separator — no match, message must pass through unchanged.
			name:        "bare vcluster name suffix without separator passes through unchanged",
			mapper:      &namespacedMapper{},
			regarding:   pod("my-pod", "default"),
			note:        "error: unknown cluster my-vc encountered",
			wantMessage: "error: unknown cluster my-vc encountered",
		},
		{
			// Cluster-scoped resources use HostNameCluster (prefixed with "vcluster-").
			// Pass 1 must replace the cluster-scoped host name using the mapper result;
			// Pass 2 does not fire because cluster-scoped objects have no namespace.
			name:        "cluster-scoped resource host name replaced in Pass 1",
			mapper:      &clusterScopedMapper{},
			regarding:   &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "my-node"}},
			note:        `Error syncing: nodes "` + translate.Default.HostNameCluster("my-node") + `" conflict`,
			wantMessage: `Error syncing: nodes "my-node" conflict`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := &captureRecorder{}
			rec := newSanitisingEventRecorder(nil, captured, tt.mapper)

			rec.Eventf(tt.regarding, nil, "Warning", "SyncError", "SyncPod", tt.note, tt.args...)

			if captured.lastMessage != tt.wantMessage {
				t.Errorf("\ngot:  %s\nwant: %s", captured.lastMessage, tt.wantMessage)
			}
			if captured.eventtype != "Warning" {
				t.Errorf("eventtype: got %q, want %q", captured.eventtype, "Warning")
			}
			if captured.reason != "SyncError" {
				t.Errorf("reason: got %q, want %q", captured.reason, "SyncError")
			}
			if captured.action != "SyncPod" {
				t.Errorf("action: got %q, want %q", captured.action, "SyncPod")
			}
		})
	}
}
