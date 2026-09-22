package telemetry

import (
	"context"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/kubernetes"
)

type noopCollector struct{}

func (n *noopCollector) RecordStart(_ context.Context, _ *config.VirtualClusterConfig) {}

func (n *noopCollector) RecordError(_ context.Context, _ *config.VirtualClusterConfig, _ ErrorSeverityType, _ error) {
}

func (n *noopCollector) Flush() {}

func (n *noopCollector) SetVirtualClient(_ kubernetes.Interface) {}

func (n *noopCollector) RecordCLI(_ *cliconfig.CLI, _ *managementv1.Self, _ error) {}
