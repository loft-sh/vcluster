package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/export_kubeconfig"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteExportKubeConfigVCluster() }

func suiteExportKubeConfigVCluster() {
	Describe("export-kubeconfig-vcluster",
		cluster.Use(clusters.ExportKubeConfigVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			export_kubeconfig.ExportKubeConfigSpec()
		},
	)
}
