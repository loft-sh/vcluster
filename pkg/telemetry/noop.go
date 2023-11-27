package telemetry

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/setup/options"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type noopCollector struct{}

func (n *noopCollector) RecordStart(ctx context.Context) {}

func (n *noopCollector) RecordError(ctx context.Context, severity ErrorSeverityType, err error) {}

func (n *noopCollector) Init(currentNamespaceConfig *rest.Config, currentNamespace string, options *options.VirtualClusterOptions) {
}

func (n *noopCollector) Flush() {}

func (n *noopCollector) SetVirtualClient(virtualClient *kubernetes.Clientset) {}
