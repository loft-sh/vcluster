package values

import (
	"errors"
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/app/create"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"k8s.io/client-go/kubernetes"
	"strings"
)

var AllowedDistros = []string{"k3s", "k0s", "k8s"}

func GetDefaultReleaseValues(client kubernetes.Interface, createOptions *create.CreateOptions, log log.Logger) (string, error) {
	if !contains(createOptions.Distro, AllowedDistros) {
		return "", fmt.Errorf("unsupported distro %s, please select one of: %s", createOptions.Distro, strings.Join(AllowedDistros, ", "))
	}

	// set correct chart name
	if createOptions.ChartName == "vcluster" && createOptions.Distro != "k3s" {
		createOptions.ChartName += "-" + createOptions.Distro
	}

	// now get the default values for the distro
	if createOptions.Distro == "k3s" {
		return getDefaultK3SReleaseValues(client, createOptions, log)
	} else if createOptions.Distro == "k0s" {
		return getDefaultK0SReleaseValues(client, createOptions, log)
	} else if createOptions.Distro == "k8s" {
		return getDefaultK8SReleaseValues(client, createOptions, log)
	}

	return "", errors.New("unrecognized distro " + createOptions.Distro)
}

func contains(needle string, haystack []string) bool {
	for _, n := range haystack {
		if needle == n {
			return true
		}
	}
	return false
}
