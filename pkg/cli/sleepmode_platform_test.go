package cli

import (
	"context"
	"errors"
	"testing"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentloftclient "github.com/loft-sh/agentapi/v4/pkg/clientset/versioned"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/auth"
	loftclient "github.com/loft-sh/api/v4/pkg/clientset/versioned"
	"github.com/loft-sh/log"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/platform"
	platformkube "github.com/loft-sh/vcluster/pkg/platform/kube"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type fakePlatformKubeClient struct {
	kubernetes.Interface
}

func (f *fakePlatformKubeClient) Loft() loftclient.Interface {
	return nil
}

func (f *fakePlatformKubeClient) Agent() agentloftclient.Interface {
	return nil
}

type fakePlatformClient struct {
	clusterClient platformkube.Interface
	clusterErr    error

	lastRestConfigHostSuffix string
	restConfig               *rest.Config
	restConfigErr            error
}

func (f *fakePlatformClient) Login(string, bool, log.Logger) error {
	return nil
}

func (f *fakePlatformClient) LoginWithAccessKey(string, string, bool) error {
	return nil
}

func (f *fakePlatformClient) Logout(context.Context) error {
	return nil
}

func (f *fakePlatformClient) Self() *managementv1.Self {
	return nil
}

func (f *fakePlatformClient) RefreshSelf(context.Context) error {
	return nil
}

func (f *fakePlatformClient) Management() (platformkube.Interface, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePlatformClient) Cluster(string) (platformkube.Interface, error) {
	return f.clusterClient, f.clusterErr
}

func (f *fakePlatformClient) VirtualCluster(string, string, string) (platformkube.Interface, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePlatformClient) ManagementConfig() (*rest.Config, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePlatformClient) RestConfig(hostSuffix string) (*rest.Config, error) {
	f.lastRestConfigHostSuffix = hostSuffix
	return f.restConfig, f.restConfigErr
}

func (f *fakePlatformClient) Config() *cliconfig.CLI {
	return nil
}

func (f *fakePlatformClient) Save() error {
	return nil
}

func (f *fakePlatformClient) Version() (*auth.Version, error) {
	return &auth.Version{}, nil
}

var _ platform.Client = &fakePlatformClient{}

func TestPausePlatformWorkloadSleepModeIfConfiguredSetsForceDuration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clusterClient := &fakePlatformKubeClient{Interface: k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"config.yaml": []byte("sleep:\n  auto:\n    afterInactivity: 1h\n"),
		},
	})}

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Spec.ClusterRef.Cluster = "host-cluster"
	virtualClusterInstance.Spec.ClusterRef.Namespace = "test-ns"

	used, err := pausePlatformWorkloadSleepModeIfConfigured(ctx, &fakePlatformClient{clusterClient: clusterClient}, "test-project", 600, log.GetInstance(), "test", virtualClusterInstance)
	assert.NilError(t, err)
	assert.Check(t, used)

	secret, err := clusterClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
	assert.Equal(t, secret.Annotations[clusterv1.SleepModeForceDurationAnnotation], "600")
	assert.Assert(t, secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation] != "")
}

func TestResumePlatformWorkloadSleepModeIfConfiguredClearsForceState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clusterClient := &fakePlatformKubeClient{Interface: k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeForceAnnotation:         "true",
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
				clusterv1.SleepModeForceDurationAnnotation: "600",
			},
		},
	})}

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Spec.ClusterRef.Cluster = "host-cluster"
	virtualClusterInstance.Spec.ClusterRef.Namespace = "test-ns"

	used, err := resumePlatformWorkloadSleepModeIfConfigured(ctx, &fakePlatformClient{clusterClient: clusterClient}, "test-project", log.GetInstance(), "test", virtualClusterInstance)
	assert.NilError(t, err)
	assert.Check(t, used)

	secret, err := clusterClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)
	_, hasForce := secret.Annotations[clusterv1.SleepModeForceAnnotation]
	assert.Assert(t, !hasForce)
	_, hasSleepType := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
	_, hasSleepingSince := secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	assert.Assert(t, !hasSleepingSince)
	_, hasForceDuration := secret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
	assert.Assert(t, !hasForceDuration)
	assert.Assert(t, secret.Annotations[clusterv1.SleepModeLastActivityAnnotation] != "")
}

func TestPausePlatformStandaloneIfConfiguredUsesRenderedValuesAndResolvedProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sentinelErr := errors.New("rest config error")
	platformClient := &fakePlatformClient{restConfigErr: sentinelErr}

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Status.VirtualCluster = &storagev1.VirtualClusterTemplateDefinition{}
	virtualClusterInstance.Status.VirtualCluster.HelmRelease.Values = "sleep:\n  auto:\n    afterInactivity: 1h\n"

	used, err := pausePlatformStandaloneIfConfigured(ctx, platformClient, "resolved/project", 900, log.GetInstance(), "test", virtualClusterInstance)
	assert.Check(t, used)
	assert.ErrorContains(t, err, sentinelErr.Error())
	assert.Equal(t, platformClient.lastRestConfigHostSuffix, "/kubernetes/project/resolved%2Fproject/virtualcluster/test")
}

func TestResumePlatformStandaloneIfConfiguredUsesRenderedValuesAndResolvedProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sentinelErr := errors.New("rest config error")
	platformClient := &fakePlatformClient{restConfigErr: sentinelErr}

	virtualClusterInstance := &managementv1.VirtualClusterInstance{}
	virtualClusterInstance.Name = "test"
	virtualClusterInstance.Spec.Standalone = true
	virtualClusterInstance.Status.VirtualCluster = &storagev1.VirtualClusterTemplateDefinition{}
	virtualClusterInstance.Status.VirtualCluster.HelmRelease.Values = "sleep:\n  auto:\n    afterInactivity: 1h\n"

	used, err := resumePlatformStandaloneIfConfigured(ctx, platformClient, "resolved/project", log.GetInstance(), "test", virtualClusterInstance)
	assert.Check(t, used)
	assert.ErrorContains(t, err, sentinelErr.Error())
	assert.Equal(t, platformClient.lastRestConfigHostSuffix, "/kubernetes/project/resolved%2Fproject/virtualcluster/test")
}
