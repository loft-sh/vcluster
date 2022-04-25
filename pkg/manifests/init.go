package manifests

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/client-go/rest"
)

const (
	InitManifestRelativePath = "init/initmanifests.yaml"
)

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

	return buf.Bytes(), nil
}
