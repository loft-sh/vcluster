package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestProToVClustersDisplaysHostWorkloadSleep(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clusterClient := &fakePlatformKubeClient{Interface: k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-release",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation: clusterv1.SleepTypeForced,
			},
		},
	})}

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Name = "instance-name"
	virtualClusterInstance.Spec.ClusterRef.Cluster = "host-cluster"
	virtualClusterInstance.Spec.ClusterRef.Namespace = "test-ns"
	virtualClusterInstance.Spec.ClusterRef.VirtualCluster = "release"
	virtualClusterInstance.Status.Phase = storagev1.InstanceReady

	output := proToVClusters(ctx, &fakePlatformClient{clusterClient: clusterClient}, []*platform.VirtualClusterInstanceProject{{
		VirtualCluster: virtualClusterInstance,
		Project:        &managementv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "test-project"}},
	}}, "")

	assert.Equal(t, len(output), 1)
	assert.Equal(t, output[0].Status, string(find.StatusWorkloadSleeping))
}

func TestProToVClustersDisplaysStandaloneWorkloadSleep(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	gotSleepModeIgnoreHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/default/secrets/"+sleepmode.StandaloneSleepSecretName {
			http.NotFound(w, r)
			return
		}
		gotSleepModeIgnoreHeader = r.Header.Get("X-Sleep-Mode-Ignore")
		if gotSleepModeIgnoreHeader != "true" {
			http.Error(w, "missing sleep mode ignore header", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      sleepmode.StandaloneSleepSecretName,
				Namespace: "default",
				Annotations: map[string]string{
					clusterv1.SleepModeSleepTypeAnnotation: clusterv1.SleepTypeForced,
				},
			},
		})
	}))
	defer server.Close()

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Name = "standalone-name"
	virtualClusterInstance.Spec.Standalone = true
	virtualClusterInstance.Status.Phase = storagev1.InstanceReady

	output := proToVClusters(ctx, &fakePlatformClient{restConfig: &rest.Config{Host: server.URL}}, []*platform.VirtualClusterInstanceProject{{
		VirtualCluster: virtualClusterInstance,
		Project:        &managementv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "test-project"}},
	}}, "")

	assert.Equal(t, len(output), 1)
	assert.Equal(t, output[0].Status, string(find.StatusWorkloadSleeping))
	assert.Equal(t, gotSleepModeIgnoreHeader, "true")
}
