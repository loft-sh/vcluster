package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/third_party/vcluster"
)

// VClusterSetup contains all the clients and configuration for a vcluster test
type VClusterSetup struct {
	Cluster        *vcluster.Cluster
	RestConfig     *rest.Config
	KlientClient   klient.Client
	KubeClient     kubernetes.Interface
	ValuesFilePath string
}

func CreateVClusterFromValues(ctx context.Context, vclusterName string, valuesYAML string) *VClusterSetup {
	setup := &VClusterSetup{}

	// Write values to a temporary file
	valuesFile := setup.createTempValuesFile(vclusterName, valuesYAML)

	// Clean up the temporary file
	e2e.DeferCleanup(func(ctx context.Context) {
		By("Removing temp vcluster.yaml file")
		_ = os.Remove(valuesFile)
	})

	// Create vcluster instance
	setup.Cluster = vcluster.NewCluster(vclusterName)

	// Create the vcluster with the values file
	_, err := setup.Cluster.CreateWithConfig(ctx, valuesFile)
	Expect(err).NotTo(HaveOccurred(), "Failed to create vcluster")

	// Get the kubeconfig and create clients
	setup.RestConfig = GetRestConfig(setup.Cluster)
	setup.KlientClient = GetKlientClient(setup.RestConfig)
	setup.KubeClient = GetKubernetesClient(setup.RestConfig)

	// Set current cluster in context
	cluster.SetCurrentCluster(vclusterName)(ctx)

	// Remove vcluster after test
	e2e.DeferCleanup(func(ctx context.Context) {
		By("Removing vcluster")
		_ = setup.Cluster.Destroy(ctx)
	})
	return setup
}

func (s *VClusterSetup) createTempValuesFile(vclusterName string, valuesYAML string) string {
	By("Creating temp vcluster.yaml file")
	tmpDir := os.TempDir()
	valuesFile := filepath.Join(tmpDir, fmt.Sprintf("vcluster-values-%s.yaml", vclusterName))
	err := os.WriteFile(valuesFile, []byte(valuesYAML), 0644)
	Expect(err).NotTo(HaveOccurred(), "Failed to write vcluster values file")
	s.ValuesFilePath = valuesFile
	return valuesFile
}

// CreateVClusterFromFile creates a vcluster using a YAML values file path
func CreateVClusterFromFile(ctx context.Context, vclusterName string, valuesFilePath string) *VClusterSetup {
	setup := &VClusterSetup{}

	// Create vcluster instance
	setup.Cluster = vcluster.NewCluster(vclusterName)

	// Create the vcluster with the values file
	By("Creating vcluster")
	_, err := setup.Cluster.CreateWithConfig(ctx, valuesFilePath)
	Expect(err).NotTo(HaveOccurred(), "Failed to create vcluster")

	// Get the kubeconfig and create clients
	setup.RestConfig = GetRestConfig(setup.Cluster)
	setup.KlientClient = GetKlientClient(setup.RestConfig)
	setup.KubeClient = GetKubernetesClient(setup.RestConfig)

	// Set current cluster in context
	cluster.SetCurrentCluster(vclusterName)(ctx)

	// Setup cleanup
	e2e.DeferCleanup(func(ctx context.Context) {
		_ = setup.Cluster.Destroy(ctx)
	})

	return setup
}

// WaitForControlPlane waits for the vcluster control plane to be ready
func (s *VClusterSetup) WaitForControlPlane(ctx context.Context) {
	By("Waiting for vcluster control plane to be ready")
	err := s.Cluster.WaitForControlPlane(ctx, s.KlientClient)
	Expect(err).NotTo(HaveOccurred(), "Failed to wait for vcluster control plane")
}

// GetRestConfig creates a rest.Config from a vcluster Cluster
func GetRestConfig(vclusterCluster *vcluster.Cluster) *rest.Config {
	kubeconfigPath := vclusterCluster.GetKubeconfig()
	Expect(kubeconfigPath).NotTo(BeEmpty(), "vcluster kubeconfig path should not be empty")

	restConfig, err := conf.New(kubeconfigPath)
	Expect(err).NotTo(HaveOccurred(), "Failed to create vcluster rest config")
	Expect(restConfig).NotTo(BeNil(), "vcluster config should not be nil")

	return restConfig
}

// GetKlientClient creates a klient.Client from a rest.Config
func GetKlientClient(restConfig *rest.Config) klient.Client {
	klientClient, err := klient.New(restConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to create klient client")
	Expect(klientClient).NotTo(BeNil(), "klient client should not be nil")

	return klientClient
}

// GetKubernetesClient creates a kubernetes.Interface from a rest.Config
func GetKubernetesClient(restConfig *rest.Config) kubernetes.Interface {
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to create kubernetes client")
	Expect(kubeClient).NotTo(BeNil(), "kubernetes client should not be nil")

	return kubeClient
}
