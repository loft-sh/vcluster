package coredns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DefaultImage = "coredns/coredns:1.14.2"
	// DefaultImageRegistry is prepended to the CoreDNS image when no custom
	// image registry is configured, so the rendered image is always fully
	// qualified (registry-less names are not reliably resolved on all runtimes).
	DefaultImageRegistry = "docker.io"
	VarImage             = "IMAGE"
	VarHostDNS           = "HOST_CLUSTER_DNS"
	VarRunAsUser         = "RUN_AS_USER"
	VarRunAsNonRoot      = "RUN_AS_NON_ROOT"
	VarRunAsGroup        = "RUN_AS_GROUP"
	VarLogInDebug        = "LOG_IN_DEBUG"
	defaultUID           = int64(1001)
	defaultGID           = int64(1001)

	// VarReplicas is the replica count the coredns deployment is rendered with.
	VarReplicas = "REPLICAS"

	// initialAutoscaledReplicas is the replica count vCluster's own autoscaler
	// starts coredns with. It is what that autoscaler runs for a cluster without
	// nodes.
	initialAutoscaledReplicas = int32(1)
)

var ErrNoCoreDNSManifests = fmt.Errorf("no coredns manifests found")

// manifestOptions is what the caller of ApplyManifest knows about the cluster
// the manifests are applied to.
type manifestOptions struct {
	// liveDeployment is the coredns deployment that currently exists, or nil
	liveDeployment *appsv1.Deployment
}

// ManifestOption tells ApplyManifest about the cluster the manifests are applied to.
type ManifestOption func(options *manifestOptions)

// WithLiveDeployment passes the coredns deployment that currently exists in the
// cluster, or nil if there is none. While autoscaling is enabled, the manifest
// is rendered with the replica count of that deployment instead of
// controlPlane.coredns.deployment.replicas. Applying the manifest then leaves
// the count alone that an autoscaler has set.
func WithLiveDeployment(deployment *appsv1.Deployment) ManifestOption {
	return func(options *manifestOptions) {
		options.liveDeployment = deployment
	}
}

func ApplyManifest(ctx context.Context, config *config.Config, defaultImageRegistry string, inClusterConfig *rest.Config, serverVersion *version.Info, options ...ManifestOption) error {
	if !config.ControlPlane.CoreDNS.Enabled {
		return nil
	}

	// get the manifest variables
	vars := manifestVariables(config, defaultImageRegistry, serverVersion, options...)

	// process the corefile and manifests
	output, err := processManifests(vars, config)
	if err != nil {
		return err
	}

	return applier.ApplyManifest(ctx, inClusterConfig, output)
}

// manifestVariables returns the variables the coredns manifests are rendered with.
func manifestVariables(vConfig *config.Config, defaultImageRegistry string, serverVersion *version.Info, options ...ManifestOption) map[string]any {
	opts := &manifestOptions{}
	for _, option := range options {
		option(opts)
	}

	vars := getManifestVariables(defaultImageRegistry, serverVersion)
	vars[VarReplicas] = deploymentReplicas(vConfig.ControlPlane.CoreDNS, opts.liveDeployment)
	return vars
}

// deploymentReplicas returns the replica count the coredns deployment is rendered with.
//
// Without autoscaling it is controlPlane.coredns.deployment.replicas. With autoscaling it is
// the count of the live deployment, so that applying the manifest does not undo what an
// autoscaler has set. Without a live deployment it is the count the autoscaler takes over from.
func deploymentReplicas(coreDNS config.CoreDNS, liveDeployment *appsv1.Deployment) int32 {
	// Never create coredns without a replica. A count of 0 used to render no count at
	// all, which the API server defaults to 1.
	configured := int32(max(coreDNS.Deployment.Replicas, 1))
	if !coreDNS.Deployment.Autoscaling.Enabled {
		return configured
	}

	if liveDeployment != nil && liveDeployment.Spec.Replicas != nil {
		return *liveDeployment.Spec.Replicas
	}

	// vCluster's own autoscaler takes over from a single replica
	if coreDNS.NodeAutoscalingEnabled() {
		return initialAutoscaledReplicas
	}

	// an external autoscaler takes over from the configured count
	return configured
}

func getManifestVariables(defaultImageRegistry string, serverVersion *version.Info) map[string]any {
	var found bool
	vars := make(map[string]any)
	vars[VarImage], found = constants.CoreDNSVersionMap[fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)]
	if !found {
		vars[VarImage] = DefaultImage
	}
	if defaultImageRegistry != "" {
		vars[VarImage] = strings.TrimSuffix(defaultImageRegistry, "/") + "/" + vars[VarImage].(string)
	} else {
		// No custom registry configured: default to docker.io (ENGCP-588).
		vars[VarImage] = DefaultImageRegistry + "/" + vars[VarImage].(string)
	}
	vars[VarRunAsUser] = fmt.Sprintf("%v", GetUserID())
	vars[VarRunAsGroup] = fmt.Sprintf("%v", GetGroupID())
	if os.Getenv("DEBUG") == "true" {
		vars[VarLogInDebug] = "log"
	} else {
		vars[VarLogInDebug] = ""
	}
	vars[VarHostDNS] = getNameserver()
	return vars
}

func getNameserver() string {
	raw, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return "/etc/resolv.conf"
	}

	nameservers := GetNameservers(raw)
	if len(nameservers) == 0 {
		return "/etc/resolv.conf"
	}

	return nameservers[0]
}

// GetGroupID retrieves the current group id and if the current process is running
// as root we fallback to GID 1001
func GetGroupID() int64 {
	gid := os.Getgid()
	if gid == 0 {
		return defaultGID
	}

	return int64(gid)
}

// GetUserID retrieves the current user id and if the current process is running
// as root we fallback to UID 1001
func GetUserID() int64 {
	uid := os.Getuid()
	if uid == 0 {
		return defaultUID
	}

	return int64(uid)
}

func DeleteCoreDNSComponents(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	labelSelector := labels.FormatLabels(map[string]string{constants.CoreDNSLabelKey: constants.CoreDNSLabelValue})

	var errs []error
	errs = append(errs, client.AppsV1().Deployments(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector}))
	errs = append(errs, client.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector}))

	services, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		errs = append(errs, err)
	} else {
		if len(services.Items) != 0 {
			for _, svc := range services.Items {
				errs = append(errs, client.CoreV1().Services(namespace).Delete(ctx, svc.Name, metav1.DeleteOptions{}))
			}
		}
	}

	return errors.Join(errs...)
}
