package coredns

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"text/template"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	Namespace             = "kube-system"
	DefaultImage          = "coredns/coredns"
	ManifestsInputFolder  = "/manifests"
	ManifestRelativePath  = "coredns/coredns.yaml"
	ManifestsOutputFolder = "/tmp/manifests-to-apply"
	VarImage              = "IMAGE"
	VarRunAsUser          = "RUN_AS_USER"
	VarRunAsNonRoot       = "RUN_AS_NON_ROOT"
	VarLogInDebug         = "LOG_IN_DEBUG"
)

var CoreDNSVersionMap = map[string]string{
	"1.23": "coredns/coredns:1.8.6",
	"1.22": "coredns/coredns:1.8.4",
	"1.21": "coredns/coredns:1.8.3",
	"1.20": "coredns/coredns:1.8.0",
	"1.19": "coredns/coredns:1.6.9",
	"1.18": "coredns/coredns:1.6.9",
	"1.17": "coredns/coredns:1.6.9",
	"1.16": "coredns/coredns:1.6.3",
}

func ApplyManifest(inClusterConfig *rest.Config, serverVersion *version.Info) error {
	// create a temporary directory and file to output processed manifest to
	debugOutputFile, err := prepareManifestOutput()
	if err != nil {
		return err
	}
	defer debugOutputFile.Close()

	vars := getManifestVariables(serverVersion)
	output, err := processManifestTemplate(vars)
	if err != nil {
		return err
	}
	// write manifest into a file for easier debugging
	_, _ = debugOutputFile.Write(*output)

	return callApply(inClusterConfig, output)
}

func prepareManifestOutput() (*os.File, error) {
	manifestOutputPath := path.Join(ManifestsOutputFolder, ManifestRelativePath)
	err := os.MkdirAll(path.Dir(manifestOutputPath), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(manifestOutputPath)
}

func getManifestVariables(serverVersion *version.Info) map[string]interface{} {
	var found bool
	vars := make(map[string]interface{})
	vars[VarImage], found = CoreDNSVersionMap[fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)]
	if !found {
		vars[VarImage] = DefaultImage
	}
	vars[VarImage] = translate.DefaultImageRegistry() + vars[VarImage].(string)

	vars[VarRunAsUser] = strconv.Itoa(os.Getuid())
	if os.Getuid() == 0 {
		vars[VarRunAsNonRoot] = "false"
	} else {
		vars[VarRunAsNonRoot] = "true"
	}
	if os.Getenv("DEBUG") == "true" {
		vars[VarLogInDebug] = "log"
	} else {
		vars[VarLogInDebug] = ""
	}
	return vars
}

func processManifestTemplate(vars map[string]interface{}) (*[]byte, error) {
	manifestInputPath := path.Join(ManifestsInputFolder, ManifestRelativePath)
	manifestTemplate, err := template.ParseFiles(manifestInputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %v", manifestInputPath, err)
	}
	buf := new(bytes.Buffer)
	err = manifestTemplate.Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("manifestTemplate.Execute failed for manifest %s: %v", manifestInputPath, err)
	}
	output := buf.Bytes()
	return &output, nil
}

func callApply(inClusterConfig *rest.Config, manifest *[]byte) error {
	restMapper, err := apiutil.NewDynamicRESTMapper(inClusterConfig)
	if err != nil {
		return fmt.Errorf("unable to initialize NewDynamicRESTMapper")
	}

	a := applier.DirectApplier{}
	opts := applier.ApplierOptions{
		Manifest:   string(*manifest),
		RESTConfig: inClusterConfig,
		RESTMapper: restMapper,
	}
	return a.Apply(context.Background(), opts)
}
