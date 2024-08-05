package coredns

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
)

const (
	DefaultImage          = "coredns/coredns:1.11.3"
	ManifestRelativePath  = "coredns/coredns.yaml"
	ManifestsOutputFolder = "/tmp/manifests-to-apply"
	VarImage              = "IMAGE"
	VarHostDNS            = "HOST_CLUSTER_DNS"
	VarRunAsUser          = "RUN_AS_USER"
	VarRunAsNonRoot       = "RUN_AS_NON_ROOT"
	VarRunAsGroup         = "RUN_AS_GROUP"
	VarLogInDebug         = "LOG_IN_DEBUG"
	defaultUID            = int64(1001)
	defaultGID            = int64(1001)
)

var ErrNoCoreDNSManifests = fmt.Errorf("no coredns manifests found")

func ApplyManifest(ctx context.Context, defaultImageRegistry string, inClusterConfig *rest.Config, serverVersion *version.Info) error {
	vars := getManifestVariables(defaultImageRegistry, serverVersion)
	output, err := processManifestTemplate(vars)
	if err != nil {
		return err
	}

	// write manifest into a file for easier debugging
	if os.Getenv("DEBUG") == "true" {
		// create a temporary directory and file to output processed manifest to
		debugOutputFile, err := prepareManifestOutput()
		if err != nil {
			return err
		}
		defer debugOutputFile.Close()

		_, _ = debugOutputFile.Write(output)
	}

	return applier.ApplyManifest(ctx, inClusterConfig, output)
}

func prepareManifestOutput() (*os.File, error) {
	manifestOutputPath := path.Join(ManifestsOutputFolder, ManifestRelativePath)
	err := os.MkdirAll(path.Dir(manifestOutputPath), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(manifestOutputPath)
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

func processManifestTemplate(vars map[string]interface{}) ([]byte, error) {
	manifestInputPath := path.Join("/manifests", ManifestRelativePath)
	// check if the manifestInputPath exists
	if _, err := os.Stat(manifestInputPath); os.IsNotExist(err) {
		return nil, ErrNoCoreDNSManifests
	}
	manifestTemplate, err := template.ParseFiles(manifestInputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", manifestInputPath, err)
	}
	buf := new(bytes.Buffer)
	err = manifestTemplate.Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("manifestTemplate.Execute failed for manifest %s: %w", manifestInputPath, err)
	}
	return buf.Bytes(), nil
}
