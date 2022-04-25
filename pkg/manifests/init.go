package manifests

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InitManifestRelativePath = "init/initmanifests.yaml"
)

var LAST_APPLIED_MANIFEST_HASH string

func ApplyInitManifests(inClusterConfig *rest.Config) error {
	vars := make(map[string]interface{})
	output, err := processManifestTemplate(vars)
	if err != nil {
		return err
	}

	return applier.ApplyManifest(inClusterConfig, output)
}

func processManifestTemplate(vars map[string]interface{}) ([]byte, error) {
	manifestInputPath := path.Join(constants.ContainerManifestsFolder, InitManifestRelativePath)
	// check if the file exists, it won't in case init.manifests is null
	_, err := os.Stat(manifestInputPath)
	if err != nil {
		return nil, err
	}

	manifestTemplate, err := template.ParseFiles(manifestInputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %v", manifestInputPath, err)
	}

	buf := new(bytes.Buffer)
	err = manifestTemplate.Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("manifestTemplate.Execute failed for manifest %s: %v", manifestInputPath, err)
	}

	// update last applied manifest hash
	content, _ := os.ReadFile(manifestInputPath)
	LAST_APPLIED_MANIFEST_HASH = hexHasher(content)

	return buf.Bytes(), nil
}

func ChangeDetected(c client.Client, namespace string) (bool, error) {
	manifestInputPath := path.Join(constants.ContainerManifestsFolder, InitManifestRelativePath)

	// check if the file exists, it won't in case init.manifests is null
	_, err := os.Stat(manifestInputPath)
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(manifestInputPath)
	if err != nil {
		return false, err
	}

	currentHash := hexHasher(content)
	if currentHash != LAST_APPLIED_MANIFEST_HASH {
		return true, nil
	}

	return false, nil
}

func hexHasher(input []byte) string {
	hash := md5.Sum(input)
	return hex.EncodeToString(hash[:])
}
