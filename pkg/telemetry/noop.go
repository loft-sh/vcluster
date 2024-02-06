package telemetry

import (
	"context"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/vcluster/pkg/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type noopCollector struct{}

func (n *noopCollector) RecordStart(_ context.Context) {}

func (n *noopCollector) RecordError(_ context.Context, _ ErrorSeverityType, _ error) {}

func (n *noopCollector) Init(_ *rest.Config, _ string, _ *options.VirtualClusterOptions) {}

func (n *noopCollector) Flush() {}

func (n *noopCollector) SetVirtualClient(_ *kubernetes.Clientset) {}

func (n *noopCollector) RecordCLI(_ *managementv1.Self, _ error) {}
