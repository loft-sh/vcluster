package filters

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUpdateNodeRejectsUnmanagedVirtualNode(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)

	hostNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared-worker",
			Labels: map[string]string{
				"env": "prod",
			},
		},
	}
	if err := pClient.Create(context.Background(), hostNode); err != nil {
		t.Fatalf("create host node: %v", err)
	}

	virtualNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared-worker",
		},
	}
	if err := vClient.Create(context.Background(), virtualNode); err != nil {
		t.Fatalf("create virtual node: %v", err)
	}
	if err := vClient.Get(context.Background(), client.ObjectKey{Name: virtualNode.Name}, virtualNode); err != nil {
		t.Fatalf("get virtual node: %v", err)
	}

	updated := virtualNode.DeepCopy()
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}
	updated.Labels["evil"] = "true"
	rawObj, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("marshal node: %v", err)
	}

	_, err = updateNode(context.Background(), encoding.NewDecoder(scheme.Scheme, false), pClient, vClient, rawObj, false)
	if !kerrors.IsForbidden(err) {
		t.Fatalf("expected forbidden for unmanaged virtual node, got %v", err)
	}

	got := &corev1.Node{}
	if err := pClient.Get(context.Background(), client.ObjectKey{Name: hostNode.Name}, got); err != nil {
		t.Fatalf("get host node: %v", err)
	}
	if got.Labels["evil"] == "true" {
		t.Fatal("host node was patched from an unmanaged virtual node")
	}
	if got.Labels["env"] != "prod" {
		t.Fatalf("host node labels changed: %v", got.Labels)
	}
}

func TestUpdateNodePatchesManagedVirtualNode(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)

	hostNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "managed-worker",
			Labels: map[string]string{
				"env": "prod",
			},
		},
	}
	if err := pClient.Create(context.Background(), hostNode); err != nil {
		t.Fatalf("create host node: %v", err)
	}

	virtualNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "managed-worker",
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
				"env":                 "prod",
			},
		},
	}
	if err := vClient.Create(context.Background(), virtualNode); err != nil {
		t.Fatalf("create virtual node: %v", err)
	}
	if err := vClient.Get(context.Background(), client.ObjectKey{Name: virtualNode.Name}, virtualNode); err != nil {
		t.Fatalf("get virtual node: %v", err)
	}

	updated := virtualNode.DeepCopy()
	updated.Labels["workload"] = "gpu"
	updated.Spec.Unschedulable = true
	rawObj, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("marshal node: %v", err)
	}

	_, err = updateNode(context.Background(), encoding.NewDecoder(scheme.Scheme, false), pClient, vClient, rawObj, false)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("update managed node: %v", err)
	}

	got := &corev1.Node{}
	if err := pClient.Get(context.Background(), client.ObjectKey{Name: hostNode.Name}, got); err != nil {
		t.Fatalf("get host node: %v", err)
	}
	if got.Labels["workload"] != "gpu" {
		t.Fatalf("expected host labels to include workload=gpu, got %v", got.Labels)
	}
	if !got.Spec.Unschedulable {
		t.Fatal("expected host node to become unschedulable")
	}
}
