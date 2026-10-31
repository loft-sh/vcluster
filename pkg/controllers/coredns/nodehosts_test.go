package coredns

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testNode(name string, addresses ...corev1.NodeAddress) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NodeStatus{Addresses: addresses},
	}
}

func internalIP(ip string) corev1.NodeAddress {
	return corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip}
}

func hostname(name string) corev1.NodeAddress {
	return corev1.NodeAddress{Type: corev1.NodeHostName, Address: name}
}

func TestCompileNodeHosts(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []*corev1.Node
		expected string
	}{
		{
			name:     "single-stack node",
			nodes:    []*corev1.Node{testNode("node-a", internalIP("10.0.0.1"), hostname("node-a"))},
			expected: "10.0.0.1 node-a",
		},
		{
			name: "dual-stack node gets one line per family",
			nodes: []*corev1.Node{
				testNode("node-a", internalIP("172.25.10.26"), internalIP("fd20:14d:70d:1014::1a"), hostname("node-a")),
			},
			expected: "172.25.10.26 node-a\nfd20:14d:70d:1014::1a node-a",
		},
		{
			name: "dual-stack node with IPv6 listed first",
			nodes: []*corev1.Node{
				testNode("node-a", internalIP("fd20:14d:70d:1014::1a"), internalIP("172.25.10.26"), hostname("node-a")),
			},
			expected: "172.25.10.26 node-a\nfd20:14d:70d:1014::1a node-a",
		},
		{
			name:     "hostname address overrides node name",
			nodes:    []*corev1.Node{testNode("node-a", internalIP("10.0.0.1"), hostname("host-a"))},
			expected: "10.0.0.1 host-a",
		},
		{
			name: "node without an InternalIP is skipped",
			nodes: []*corev1.Node{
				testNode("node-a", hostname("node-a")),
				testNode("node-b", internalIP("10.0.0.2"), hostname("node-b")),
			},
			expected: "10.0.0.2 node-b",
		},
		{
			name:     "no nodes",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			assert.NilError(t, corev1.AddToScheme(scheme))
			objects := make([]client.Object, 0, len(tt.nodes))
			for _, node := range tt.nodes {
				objects = append(objects, node)
			}
			r := &NodeHostsReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
				Log:    loghelper.New("test"),
			}

			nodehosts, err := r.compileNodeHosts(context.Background())
			assert.NilError(t, err)
			assert.Equal(t, nodehosts, tt.expected)
		})
	}
}
