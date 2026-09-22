package find

import (
	"os"
	"path/filepath"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetStandaloneKubeClientConfigUsesDataDirAdminConf(t *testing.T) {
	dataDir := t.TempDir()
	kubeConfigPath := filepath.Join(dataDir, "pki", "admin.conf")
	writeTestKubeConfig(t, kubeConfigPath, "https://standalone.example.test")

	vConfig := &vclusterconfig.VirtualClusterConfig{}
	vConfig.ControlPlane.Standalone.DataDir = dataDir

	clientConfig, err := getStandaloneKubeClientConfig(vConfig)
	if err != nil {
		t.Fatalf("getStandaloneKubeClientConfig() error = %v", err)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		t.Fatalf("ClientConfig() error = %v", err)
	}
	if restConfig.Host != "https://standalone.example.test" {
		t.Fatalf("expected standalone kubeconfig host, got %q", restConfig.Host)
	}
}

func TestGetStandaloneKubeClientConfigUsesExplicitKubeConfig(t *testing.T) {
	dataDir := t.TempDir()
	defaultKubeConfigPath := filepath.Join(dataDir, "pki", "admin.conf")
	writeTestKubeConfig(t, defaultKubeConfigPath, "https://default.example.test")

	explicitKubeConfigPath := filepath.Join(t.TempDir(), "custom.conf")
	writeTestKubeConfig(t, explicitKubeConfigPath, "https://custom.example.test")

	vConfig := &vclusterconfig.VirtualClusterConfig{}
	vConfig.ControlPlane.Standalone.DataDir = dataDir
	vConfig.Experimental.VirtualClusterKubeConfig.KubeConfig = explicitKubeConfigPath

	clientConfig, err := getStandaloneKubeClientConfig(vConfig)
	if err != nil {
		t.Fatalf("getStandaloneKubeClientConfig() error = %v", err)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		t.Fatalf("ClientConfig() error = %v", err)
	}
	if restConfig.Host != "https://custom.example.test" {
		t.Fatalf("expected explicit kubeconfig host, got %q", restConfig.Host)
	}
}

func writeTestKubeConfig(t *testing.T, path, host string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("mkdir kubeconfig dir: %v", err)
	}

	err = clientcmd.WriteToFile(clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test": {Server: host, InsecureSkipTLSVerify: true},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test": {Token: "token"},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test": {Cluster: "test", AuthInfo: "test"},
		},
		CurrentContext: "test",
	}, path)
	if err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
}
