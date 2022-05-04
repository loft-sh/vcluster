package applier

import (
	"context"
	"fmt"
	"io/ioutil"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func ApplyManifestFile(inClusterConfig *rest.Config, filename string, applyOpts ...ApplyOptions) error {
	manifest, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("function ApplyManifestFile failed, unable to read %s file: %v", filename, err)
	}

	return ApplyManifest(inClusterConfig, manifest, applyOpts...)
}

func ApplyManifest(inClusterConfig *rest.Config, manifests []byte, applyOpts ...ApplyOptions) error {
	restMapper, err := apiutil.NewDynamicRESTMapper(inClusterConfig)
	if err != nil {
		return fmt.Errorf("unable to initialize NewDynamicRESTMapper")
	}

	a := DirectApplier{}
	opts := applierOptions{
		RESTMapper: restMapper,
		RESTConfig: inClusterConfig,
		Manifest:   string(manifests),
	}

	for _, aOpt := range applyOpts {
		aOpt(&opts)
	}

	return a.Apply(context.Background(), opts)
}
