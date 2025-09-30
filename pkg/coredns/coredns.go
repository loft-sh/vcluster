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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DefaultImage    = "coredns/coredns:1.12.1"
	VarImage        = "IMAGE"
	VarHostDNS      = "HOST_CLUSTER_DNS"
	VarRunAsUser    = "RUN_AS_USER"
	VarRunAsNonRoot = "RUN_AS_NON_ROOT"
	VarRunAsGroup   = "RUN_AS_GROUP"
	VarLogInDebug   = "LOG_IN_DEBUG"
	defaultUID      = int64(1001)
	defaultGID      = int64(1001)
)

var ErrNoCoreDNSManifests = fmt.Errorf("no coredns manifests found")

func ApplyManifest(ctx context.Context, config *config.Config, defaultImageRegistry string, inClusterConfig *rest.Config, serverVersion *version.Info) error {
	if !config.ControlPlane.CoreDNS.Enabled {
		return nil
	}

	// get the manifest variables
	vars := getManifestVariables(defaultImageRegistry, serverVersion)

	// process the corefile and manifests
	output, err := processManifests(vars, config)
	if err != nil {
		return err
	}

	return applier.ApplyManifest(ctx, inClusterConfig, output)
}

func getManifestVariables(defaultImageRegistry string, serverVersion *version.Info) map[string]interface{} {
	var found bool
	vars := make(map[string]interface{})
	vars[VarImage], found = constants.CoreDNSVersionMap[fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)]
	if !found {
		vars[VarImage] = DefaultImage
	}
	if defaultImageRegistry != "" {
		vars[VarImage] = strings.TrimSuffix(defaultImageRegistry, "/") + "/" + vars[VarImage].(string)
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
