package applier

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func ApplyManifestFile(ctx context.Context, inClusterConfig *rest.Config, filename string) error {
	manifest, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("function ApplyManifestFile failed, unable to read %s file: %w", filename, err)
	}

	return ApplyManifest(ctx, inClusterConfig, manifest)
}

func ApplyManifest(ctx context.Context, inClusterConfig *rest.Config, manifests []byte) error {
	httpClient, err := rest.HTTPClientFor(inClusterConfig)
	if err != nil {
		return fmt.Errorf("unable to initialize HTTPClientFor")
	}
	restMapper, err := apiutil.NewDynamicRESTMapper(inClusterConfig, httpClient)
	if err != nil {
		return fmt.Errorf("unable to initialize NewDynamicRESTMapper")
	}

	a := DirectApplier{}
	opts := Options{
		RESTMapper: restMapper,
		RESTConfig: inClusterConfig,
		Manifest:   string(manifests),
	}
	return a.Apply(ctx, opts)
}
