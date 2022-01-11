package context

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncContext struct {
	Context       context.Context
	Log           loghelper.Logger
	Name          string
	EventRecorder record.EventRecorder

	TargetNamespace string
	PhysicalClient  client.Client

	VirtualClient client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
}
