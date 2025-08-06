package deploy

import (
	_ "embed"

	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/applier"
)

var (
	//go:embed snapshot.storage.k8s.io-crds-v8.3.0.yaml
	snapshotCRDs string
)

func Deploy(ctx *synccontext.ControllerContext) error {
	// apply the manifests
	klog.Infof("Applying snapshot CustomResourceDefinitions...")
	return applier.ApplyManifest(ctx, ctx.VirtualManager.GetConfig(), []byte(snapshotCRDs))
}
