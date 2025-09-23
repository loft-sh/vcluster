package deploy

import (
	_ "embed"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/applier"
)

var (
	//go:embed snapshot.storage.k8s.io-crds-v8.3.0.yaml
	snapshotCRDs string

	//go:embed snapshot-controller-v8.3.0.yaml
	snapshotController string
)

func Deploy(ctx *synccontext.RegisterContext) error {
	// apply the volume snapshot CustomResourceDefinition manifests
	klog.Infof("Applying volume snapshot CustomResourceDefinitions...")
	err := applier.ApplyManifest(ctx, ctx.VirtualManager.GetConfig(), []byte(snapshotCRDs))
	if err != nil {
		return fmt.Errorf("failed to apply volume snapshot CustomResourceDefinitions: %w", err)
	}

	// apply the snapshot controller manifests
	klog.Infof("Applying snapshot controller...")
	err = applier.ApplyManifest(ctx, ctx.VirtualManager.GetConfig(), []byte(snapshotController))
	if err != nil {
		return fmt.Errorf("failed to apply snapshot controller: %w", err)
	}

	return nil
}
