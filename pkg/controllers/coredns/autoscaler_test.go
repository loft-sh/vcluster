package coredns

import (
	"context"
	"fmt"
	"testing"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// defaultAutoscaling mirrors the chart defaults of controlPlane.coredns.deployment.autoscaling.
var defaultAutoscaling = config.CoreDNSAutoscaling{
	Enabled:         true,
	NodesPerReplica: 16,
	MinReplicas:     2,
	MaxReplicas:     100,
}

func TestDesiredReplicas(t *testing.T) {
	tests := []struct {
		name        string
		nodes       int
		autoscaling config.CoreDNSAutoscaling
		expected    int32
	}{
		{name: "cluster without nodes runs one replica", nodes: 0, autoscaling: defaultAutoscaling, expected: 1},
		{name: "single node runs one replica, below minReplicas", nodes: 1, autoscaling: defaultAutoscaling, expected: 1},
		{name: "second node raises the count to minReplicas", nodes: 2, autoscaling: defaultAutoscaling, expected: 2},
		{name: "minReplicas holds up to two full ratios", nodes: 32, autoscaling: defaultAutoscaling, expected: 2},
		{name: "one node over the ratio adds a replica", nodes: 33, autoscaling: defaultAutoscaling, expected: 3},
		{name: "large fleet follows the ratio", nodes: 1000, autoscaling: defaultAutoscaling, expected: 63},
		{name: "maxReplicas caps the count", nodes: 5000, autoscaling: defaultAutoscaling, expected: 100},
		{
			name:        "tighter ratio for GPU fleets",
			nodes:       1000,
			autoscaling: config.CoreDNSAutoscaling{NodesPerReplica: 8, MinReplicas: 2, MaxReplicas: 200},
			expected:    125,
		},
		{
			name:        "one replica per node",
			nodes:       3,
			autoscaling: config.CoreDNSAutoscaling{NodesPerReplica: 1, MinReplicas: 1, MaxReplicas: 100},
			expected:    3,
		},
		{
			name:        "minReplicas equal to maxReplicas pins the count",
			nodes:       500,
			autoscaling: config.CoreDNSAutoscaling{NodesPerReplica: 16, MinReplicas: 4, MaxReplicas: 4},
			expected:    4,
		},
		{
			name:        "zero ratio does not divide by zero",
			nodes:       5,
			autoscaling: config.CoreDNSAutoscaling{MinReplicas: 2, MaxReplicas: 100},
			expected:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, DesiredReplicas(tt.nodes, tt.autoscaling), tt.expected)
		})
	}
}

func testDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: Namespace, Name: DeploymentName},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	}
}

func TestAutoscalerReconcile(t *testing.T) {
	tests := []struct {
		name     string
		nodes    int
		replicas int32
		expected int32
	}{
		{name: "scales up when nodes joined", nodes: 40, replicas: 1, expected: 3},
		{name: "scales down when nodes left", nodes: 3, replicas: 5, expected: 2},
		{name: "scales down to one replica on the last node", nodes: 1, replicas: 2, expected: 1},
		{name: "undoes a count set by something else", nodes: 40, replicas: 10, expected: 3},
		{name: "caps at maxReplicas", nodes: 2000, replicas: 20, expected: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{testDeployment(tt.replicas)}
			for i := range tt.nodes {
				objects = append(objects, testNode(fmt.Sprintf("node-%d", i)))
			}
			r := newTestAutoscaler(t, objects...)

			_, err := r.Reconcile(context.Background(), ctrl.Request{})
			assert.NilError(t, err)

			deployment := &appsv1.Deployment{}
			assert.NilError(t, r.Get(context.Background(), client.ObjectKeyFromObject(testDeployment(0)), deployment))
			assert.Equal(t, *deployment.Spec.Replicas, tt.expected)
		})
	}
}

func TestAutoscalerReconcileLeavesDesiredCountAlone(t *testing.T) {
	r := newTestAutoscaler(t, testDeployment(2), testNode("node-a"), testNode("node-b"))
	key := client.ObjectKeyFromObject(testDeployment(0))

	before := &appsv1.Deployment{}
	assert.NilError(t, r.Get(context.Background(), key, before))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	assert.NilError(t, err)

	after := &appsv1.Deployment{}
	assert.NilError(t, r.Get(context.Background(), key, after))
	assert.Equal(t, after.ResourceVersion, before.ResourceVersion, "a deployment that already has the desired count must not be written")
}

func TestAutoscalerReconcileWithoutDeployment(t *testing.T) {
	r := newTestAutoscaler(t, testNode("node-a"), testNode("node-b"))

	result, err := r.Reconcile(context.Background(), ctrl.Request{})
	assert.NilError(t, err)
	assert.Equal(t, result, ctrl.Result{}, "the deployment's create event triggers the next reconcile, no requeue needed")
}

func TestAutoscalerReconcileIgnoresOtherDeployments(t *testing.T) {
	other := testDeployment(1)
	other.Name = "metrics-server"
	r := newTestAutoscaler(t, other, testDeployment(1), testNode("node-a"), testNode("node-b"))

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	assert.NilError(t, err)

	untouched := &appsv1.Deployment{}
	assert.NilError(t, r.Get(context.Background(), client.ObjectKeyFromObject(other), untouched))
	assert.Equal(t, *untouched.Spec.Replicas, int32(1))
}

func newTestAutoscaler(t *testing.T, objects ...client.Object) *AutoscalerReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	assert.NilError(t, corev1.AddToScheme(scheme))
	assert.NilError(t, appsv1.AddToScheme(scheme))

	return &AutoscalerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
		Log:    loghelper.New("test"),
		Config: defaultAutoscaling,
	}
}
