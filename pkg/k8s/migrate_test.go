package k8s

import (
	"context"
	"slices"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func node(name string, finalizers ...string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Finalizers: finalizers,
		},
	}
}

func TestRemoveK3sNodeFinalizers(t *testing.T) {
	testCases := []struct {
		name               string
		nodes              []runtime.Object
		expectedFinalizers map[string][]string
	}{
		{
			name:  "removes the k3s node finalizer",
			nodes: []runtime.Object{node("node-1", "wrangler.cattle.io/node")},
			expectedFinalizers: map[string][]string{
				"node-1": nil,
			},
		},
		{
			// Narrowed to the exact k3s finalizer: other wrangler.cattle.io/* finalizers (e.g. ones a
			// controller like Rancher might add) must be left untouched.
			name: "removes only the k3s node finalizer, keeps other finalizers",
			nodes: []runtime.Object{
				node("node-1", "wrangler.cattle.io/node", "example.com/keep", "wrangler.cattle.io/managed-etcd-controller"),
			},
			expectedFinalizers: map[string][]string{
				"node-1": {"example.com/keep", "wrangler.cattle.io/managed-etcd-controller"},
			},
		},
		{
			name: "leaves nodes without the k3s finalizer untouched",
			nodes: []runtime.Object{
				node("node-1", "example.com/keep"),
				node("node-2"),
			},
			expectedFinalizers: map[string][]string{
				"node-1": {"example.com/keep"},
				"node-2": nil,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeClient := testingutil.NewFakeClient(scheme.Scheme, testCase.nodes...)

			err := removeK3sNodeFinalizers(ctx, fakeClient)
			assert.NilError(t, err)

			for name, expected := range testCase.expectedFinalizers {
				gotNode := &corev1.Node{}
				err := fakeClient.Get(ctx, types.NamespacedName{Name: name}, gotNode)
				assert.NilError(t, err)
				assert.DeepEqual(t, expected, gotNode.Finalizers)
			}
		})
	}
}

func TestCleanupK3sNodeFinalizers(t *testing.T) {
	const namespace = "test-ns"
	options := &config.VirtualClusterConfig{}
	options.Name = "test"
	secretName := k3sMigrationSecretName(options)

	migrationSecret := func(migrated, cleaned bool) *corev1.Secret {
		annotations := map[string]string{}
		if migrated {
			annotations[migratedFromK3sAnnotation] = "true"
		}
		if cleaned {
			annotations[migratedFromK3sNodesCleanedAnnotation] = "true"
		}
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace, Annotations: annotations},
		}
	}

	t.Run("strips the finalizer and records the cleaned marker on first run", func(t *testing.T) {
		ctx := context.Background()
		hostClient := k8sfake.NewSimpleClientset(migrationSecret(true, false))
		vClient := testingutil.NewFakeClient(scheme.Scheme, node("node-1", "wrangler.cattle.io/node"))

		err := CleanupK3sNodeFinalizers(ctx, hostClient, namespace, vClient, options)
		assert.NilError(t, err)

		gotNode := &corev1.Node{}
		assert.NilError(t, vClient.Get(ctx, types.NamespacedName{Name: "node-1"}, gotNode))
		assert.Assert(t, !slices.Contains(gotNode.Finalizers, k3sNodeFinalizer), "finalizer should be stripped")

		gotSecret, err := hostClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, gotSecret.Annotations[migratedFromK3sNodesCleanedAnnotation], "true")
	})

	t.Run("does nothing when the cleaned marker is already set", func(t *testing.T) {
		ctx := context.Background()
		hostClient := k8sfake.NewSimpleClientset(migrationSecret(true, true))
		vClient := testingutil.NewFakeClient(scheme.Scheme, node("node-1", "wrangler.cattle.io/node"))

		err := CleanupK3sNodeFinalizers(ctx, hostClient, namespace, vClient, options)
		assert.NilError(t, err)

		gotNode := &corev1.Node{}
		assert.NilError(t, vClient.Get(ctx, types.NamespacedName{Name: "node-1"}, gotNode))
		assert.Assert(t, slices.Contains(gotNode.Finalizers, k3sNodeFinalizer), "finalizer should be left in place")
	})

	t.Run("does nothing when the cluster was not migrated from k3s", func(t *testing.T) {
		ctx := context.Background()
		hostClient := k8sfake.NewSimpleClientset(migrationSecret(false, false))
		vClient := testingutil.NewFakeClient(scheme.Scheme, node("node-1", "wrangler.cattle.io/node"))

		err := CleanupK3sNodeFinalizers(ctx, hostClient, namespace, vClient, options)
		assert.NilError(t, err)

		gotNode := &corev1.Node{}
		assert.NilError(t, vClient.Get(ctx, types.NamespacedName{Name: "node-1"}, gotNode))
		assert.Assert(t, slices.Contains(gotNode.Finalizers, k3sNodeFinalizer), "finalizer should be left in place")
	})
}
