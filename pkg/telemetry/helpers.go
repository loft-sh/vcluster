package telemetry

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/denisbrodbeck/machineid"
	vcontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/telemetry/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SyncerVersion                                        = "dev"
	cachedUID                                            = ""
	cachedSyncerFlags                                    = ""
	cachedHostKubernetesVersion *types.KubernetesVersion = nil
	cachedVclusterServiceType                            = ""
)

const (
	ConfigEnvVar = "VCLUSTER_TELEMETRY_CONFIG"

	// hashingKey is a random string used for hashing the CreatorUID.
	// It shouldn't be changed after the release.
	hashingKey = "1f1uR6nVryzFEaAm87Ra"
)

// getSyncerUID provides instance UID based on the UID of the PVC or Service
func getSyncerUID(ctx context.Context, c *kubernetes.Clientset, vclusterNamespace string, options *vcontext.VirtualClusterOptions) string {
	if cachedUID != "" {
		return cachedUID
	}

	if c == nil || options == nil {
		return ""
	}

	o, err := getUniqueSyncerObject(ctx, c, vclusterNamespace, options)
	if err == nil {
		cachedUID = string(o.GetUID())
		return cachedUID
	}

	return ""
}

// returns a Kubernetes resource that can be used to uniquely identify this syncer instance - PVC or Service
func getUniqueSyncerObject(ctx context.Context, c *kubernetes.Clientset, vclusterNamespace string, options *vcontext.VirtualClusterOptions) (client.Object, error) {
	// we primarily use PVC as the source of vcluster instance UID
	pvc, err := c.CoreV1().PersistentVolumeClaims(vclusterNamespace).Get(ctx, fmt.Sprintf("data-%s-0", translate.Suffix), metav1.GetOptions{})
	if err == nil {
		return pvc, nil
	}
	if !kerrors.IsNotFound(err) {
		return nil, err
	}

	// If vcluster PVC doesn't exist we try to get UID from the vcluster Service
	if options.ServiceName == "" {
		return nil, fmt.Errorf("getUniqueSyncerObject failed - PVC was not found and options.ServiceName is empty")
	}
	service, err := c.CoreV1().Services(vclusterNamespace).Get(ctx, options.ServiceName, metav1.GetOptions{})
	if err == nil {
		return service, nil
	}
	return nil, err
}

func getSyncerFlags(startCommand *cobra.Command, options *vcontext.VirtualClusterOptions) string {
	if cachedSyncerFlags != "" {
		return cachedSyncerFlags
	}

	if startCommand == nil || options == nil {
		return ""
	}

	setFlags := map[string]bool{}
	startCommand.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			setFlags[f.Name] = true
		}
	})

	o, err := json.Marshal(types.SyncerFlags{
		SetFlags:    setFlags,
		Controllers: options.Controllers,
	})
	if err != nil {
		return ""
	}

	cachedSyncerFlags = string(o)
	return cachedSyncerFlags
}

func getVirtualKubernetesVersion(c *kubernetes.Clientset) *types.KubernetesVersion {
	if c == nil {
		return nil
	}

	vi, _ := c.Discovery().ServerVersion()
	return toKubernetesVersion(vi)
}

func getHostKubernetesVersion(c *kubernetes.Clientset) *types.KubernetesVersion {
	if cachedHostKubernetesVersion != nil {
		return cachedHostKubernetesVersion
	}

	if c == nil {
		return nil
	}

	vi, err := c.Discovery().ServerVersion()
	if err == nil {
		cachedHostKubernetesVersion = toKubernetesVersion(vi)
	}
	return cachedHostKubernetesVersion
}

func toKubernetesVersion(vi *version.Info) *types.KubernetesVersion {
	if vi == nil {
		return nil
	}
	return &types.KubernetesVersion{
		Major:      vi.Major,
		Minor:      vi.Minor,
		GitVersion: vi.GitVersion,
	}
}

func getVclusterServiceType(ctx context.Context, c *kubernetes.Clientset, vclusterNamespace string, options *vcontext.VirtualClusterOptions) string {
	if cachedVclusterServiceType != "" {
		return cachedVclusterServiceType
	}

	if c == nil || options == nil {
		return ""
	}

	// Let's first check if a separate LoadBalancer Service is created
	service, err := c.CoreV1().Services(vclusterNamespace).Get(ctx, fmt.Sprintf("%s-lb", translate.Suffix), metav1.GetOptions{})
	if err == nil {
		cachedVclusterServiceType = string(service.Spec.Type)
		return cachedVclusterServiceType
	}

	// otherwise check the type of the usual vcluster Service
	service, err = c.CoreV1().Services(vclusterNamespace).Get(ctx, options.ServiceName, metav1.GetOptions{})
	if err == nil {
		cachedVclusterServiceType = string(service.Spec.Type)
	}
	return cachedVclusterServiceType
}

// Gets machine ID and encodes it together with users $HOME path and extra key to protect privacy.
// Returns a hex-encoded string.
func GetInstanceCreatorUID() string {
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
	mac.Write([]byte(hashingKey))
	mac.Write([]byte(home))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
