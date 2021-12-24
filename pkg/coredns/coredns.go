package coredns

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/loft-sh/vcluster/pkg/util/applier"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	VarDNSPolicy          = "DNS_POLICY"
	DefaultDNSPolicy      = string(corev1.DNSDefault)
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

func GetPodSelector() labels.Selector {
	return labels.SelectorFromSet(map[string]string{"k8s-app": "kube-dns"})
}

func ApplyManifest(inClusterConfig *rest.Config, serverVersion *version.Info, cliVariables []string) error {
	// create a temporary directory and file to output processed manifest to
	debugOutputFile, err := prepareManifestOutput()
	if err != nil {
		return err
	}
	defer debugOutputFile.Close()

	cliVarsParsed, err := parseCliVars(cliVariables)
	if err != nil {
		return err
	}

	vars := getManifestVariables(serverVersion, cliVarsParsed)
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

func parseCliVars(cliVariables []string) (map[string]string, error) {
	vars := map[string]string{}
	for _, v := range cliVariables {
		i := strings.SplitN(strings.TrimSpace(v), "=", 2)
		if len(i) != 2 {
			return nil, fmt.Errorf("error parsing coredns-var '%s': bad format, expected VAR_NAME=VALUE", v)
		}
		vars[i[0]] = i[1]
	}
	return vars, nil
}

func getManifestVariables(serverVersion *version.Info, cliVariables map[string]string) map[string]interface{} {
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

	vars[VarDNSPolicy] = DefaultDNSPolicy

	for k, v := range cliVariables {
		vars[k] = v
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
