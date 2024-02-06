package telemetry

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/denisbrodbeck/machineid"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/options"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	homedir "github.com/mitchellh/go-homedir"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SyncerVersion = "dev"
)

func getChartInfo(ctx context.Context, hostClient *kubernetes.Clientset, vClusterNamespace string) (*ChartInfo, error) {
	if hostClient == nil {
		return nil, fmt.Errorf("host client is empty")
	}

	release, err := helm.NewSecrets(hostClient).Get(ctx, translate.VClusterName, vClusterNamespace)
	if err != nil {
		return nil, err
	} else if release == nil {
		return &ChartInfo{}, nil
	} else if kerrors.IsNotFound(err) {
		return &ChartInfo{}, nil
	}

	if release.Config == nil {
		release.Config = map[string]interface{}{}
	}

	name := "unknown"
	chartVersion := ""
	if release.Chart != nil && release.Chart.Metadata != nil && release.Chart.Metadata.Name != "" {
		name = release.Chart.Metadata.Name
		chartVersion = release.Chart.Metadata.Version
	}

	return &ChartInfo{
		Name:    name,
		Version: chartVersion,
		Values:  release.Config,
	}, nil
}

// getVClusterID provides instance ID based on the UID of the service
func getVClusterID(ctx context.Context, hostClient *kubernetes.Clientset, vClusterNamespace string, options *options.VirtualClusterOptions) (string, error) {
	if hostClient == nil || options == nil {
		return "", fmt.Errorf("kubernetes client or options are nil")
	}

	o, err := getUniqueSyncerObject(ctx, hostClient, vClusterNamespace, options)
	if err != nil {
		return "", err
	}

	return string(o.GetUID()), nil
}

// returns a Kubernetes resource that can be used to uniquely identify this syncer instance - PVC or Service
func getUniqueSyncerObject(ctx context.Context, c *kubernetes.Clientset, vClusterNamespace string, options *options.VirtualClusterOptions) (client.Object, error) {
	// If vCluster PVC doesn't exist we try to get UID from the vCluster Service
	if options.ServiceName == "" {
		return nil, fmt.Errorf("getUniqueSyncerObject failed - options.ServiceName is empty")
	}

	service, err := c.CoreV1().Services(vClusterNamespace).Get(ctx, options.ServiceName, metav1.GetOptions{})
	if err == nil {
		return service, nil
	}

	return nil, err
}

func getKubernetesVersion(c *kubernetes.Clientset) (*KubernetesVersion, error) {
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
func GetPlatformUserID(self *managementv1.Self) string {
	if cliconfig.GetConfig(log.Discard).TelemetryDisabled || self == nil {
		return ""
	}
	platformID := self.Status.Subject
	if self.Status.User != nil && self.Status.User.Email != "" {
		platformID = self.Status.User.Email
	}
	return platformID
}

// GetPlatformInstanceID returns the loft instance id
func GetPlatformInstanceID(self *managementv1.Self) string {
	if cliconfig.GetConfig(log.Discard).TelemetryDisabled || self == nil {
		return ""
	}

	return self.Status.InstanceID
}

// GetMachineID retrieves machine ID and encodes it together with users $HOME path and
// extra key to protect privacy. Returns a hex-encoded string.
func GetMachineID(log log.Logger) string {
	if cliconfig.GetConfig(log).TelemetryDisabled {
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
