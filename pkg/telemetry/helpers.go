package telemetry

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/denisbrodbeck/machineid"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	homedir "github.com/mitchellh/go-homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SyncerVersion = "dev"
)

// getVClusterID provides instance ID based on the UID of the service
func getVClusterID(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace, vClusterService string) (string, error) {
	if hostClient == nil || vClusterService == "" {
		return "", fmt.Errorf("kubernetes client or service is undefined")
	}

	o, err := getUniqueSyncerObject(ctx, hostClient, vClusterNamespace, vClusterService)
	if err != nil {
		return "", err
	}

	return string(o.GetUID()), nil
}

// getVClusterCreationTimestamp returns the creation timestamp of the vCluster service
func getVClusterCreationTimestamp(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace, vClusterService string) (int64, error) {
	if hostClient == nil || vClusterService == "" {
		return 0, fmt.Errorf("kubernetes client or service is undefined")
	}

	o, err := getUniqueSyncerObject(ctx, hostClient, vClusterNamespace, vClusterService)
	if err != nil {
		return 0, err
	}

	return o.GetCreationTimestamp().Unix(), nil
}

// returns a Kubernetes resource that can be used to uniquely identify this syncer instance - PVC or Service
func getUniqueSyncerObject(ctx context.Context, c kubernetes.Interface, vClusterNamespace string, serviceName string) (client.Object, error) {
	// If vCluster PVC doesn't exist we try to get UID from the vCluster Service
	if serviceName == "" {
		return nil, fmt.Errorf("getUniqueSyncerObject failed - options.ServiceName is empty")
	}

	service, err := c.CoreV1().Services(vClusterNamespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err == nil {
		return service, nil
	}

	return nil, err
}

func getKubernetesVersion(c kubernetes.Interface) (*KubernetesVersion, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	vi, err := c.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error retrieving version: %w", err)
	}

	return toKubernetesVersion(vi), nil
}

func toKubernetesVersion(vi *version.Info) *KubernetesVersion {
	if vi == nil {
		return nil
	}
	return &KubernetesVersion{
		Major:      vi.Major,
		Minor:      vi.Minor,
		GitVersion: vi.GitVersion,
	}
}

// GetPlatformUserID returns the loft instance id
func GetPlatformUserID(cliConfig *config.CLI, self *managementv1.Self) string {
	if cliConfig.TelemetryDisabled || self == nil {
		return ""
	}
	platformID := self.Status.Subject
	if self.Status.User != nil && self.Status.User.Email != "" {
		platformID = self.Status.User.Email
	}
	return platformID
}

// GetPlatformInstanceID returns the loft instance id
func GetPlatformInstanceID(cliConfig *config.CLI, self *managementv1.Self) string {
	if cliConfig.TelemetryDisabled || self == nil {
		return ""
	}

	return self.Status.InstanceID
}

// GetMachineID retrieves machine ID and encodes it together with users $HOME path and
// extra key to protect privacy. Returns a hex-encoded string.
func GetMachineID(cliConfig *config.CLI) string {
	if cliConfig.TelemetryDisabled {
		return ""
	}

	id, err := machineid.ID()
	if err != nil {
		id = "error"
	}

	// get $HOME to distinguish two users on the same machine
	// will be hashed later together with the ID
	home, err := homedir.Dir()
	if err != nil {
		home = "error"
	}

	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
