package find

import (
	"context"
	"errors"
	"testing"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentloftclient "github.com/loft-sh/agentapi/v4/pkg/clientset/versioned"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// fakeKubeClient adapts a fake kubernetes clientset to the platform kube.Interface
// expected by getVCluster. Loft()/Agent() are never used on the status code path.
type fakeKubeClient struct {
	kubernetes.Interface
}

func (fakeKubeClient) Loft() loftclient.Interface       { return nil }
func (fakeKubeClient) Agent() agentloftclient.Interface { return nil }

func TestGetVClusterStatus(t *testing.T) {
	const (
		release = "my-vc"
		ns      = "my-ns"
	)

	statefulSet := func(mutate func(sts *appsv1.StatefulSet)) *appsv1.StatefulSet {
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: release, Namespace: ns},
			Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(1))},
		}
		if mutate != nil {
			mutate(sts)
		}
		return sts
	}

	runningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      release + "-0",
			Namespace: ns,
			Labels:    map[string]string{"app": "vcluster", "release": release},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}

	sleepConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "vc-config-" + release,
			Namespace:   ns,
			Annotations: map[string]string{clusterv1.SleepModeSleepTypeAnnotation: "all"},
		},
	}

	tests := []struct {
		name    string
		object  *appsv1.StatefulSet
		objects []runtime.Object
		want    Status
	}{
		{
			name: "paused via annotation",
			object: statefulSet(func(sts *appsv1.StatefulSet) {
				sts.Annotations = map[string]string{constants.PausedAnnotation(false): "true"}
			}),
			want: StatusPaused,
		},
		{
			name:   "paused via sleep-mode label",
			object: statefulSet(func(sts *appsv1.StatefulSet) { sts.Labels = map[string]string{sleepmode.Label: "true"} }),
			want:   StatusPaused,
		},
		{
			name:    "workloads sleeping via config secret",
			object:  statefulSet(nil),
			objects: []runtime.Object{sleepConfigSecret},
			want:    StatusWorkloadSleeping,
		},
		{
			name:   "scaled down when no pods and zero replicas",
			object: statefulSet(func(sts *appsv1.StatefulSet) { sts.Spec.Replicas = new(int32(0)) }),
			want:   StatusScaledDown,
		},
		{
			name:    "running when a pod is running",
			object:  statefulSet(nil),
			objects: []runtime.Object{runningPod},
			want:    Status("Running"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeKubeClient{Interface: fake.NewSimpleClientset(tt.objects...)}
			vc, err := getVCluster(context.Background(), tt.object, "", release, client, nil)
			if err != nil {
				t.Fatalf("getVCluster() error = %v", err)
			}
			if vc.Status != tt.want {
				t.Errorf("status = %q, want %q", vc.Status, tt.want)
			}
		})
	}
}

func TestStandaloneStatus(t *testing.T) {
	errSystemctl := errors.New("systemctl: command not found")

	tests := []struct {
		name   string
		output string
		err    error
		want   Status
	}{
		{"active service is running", "active\n", nil, StatusRunning},
		{"inactive service is unknown", "inactive\n", nil, StatusUnknown},
		{"failed service is unknown", "failed\n", nil, StatusUnknown},
		{"command error is unknown", "", errSystemctl, StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := standaloneStatus([]byte(tt.output), tt.err); got != tt.want {
				t.Errorf("standaloneStatus(%q, %v) = %q, want %q", tt.output, tt.err, got, tt.want)
			}
		})
	}
}
