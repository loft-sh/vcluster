package context

import (
	"context"

	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncContext struct {
	Context context.Context
	Log     loghelper.Logger

	TargetNamespace string
	PhysicalClient  client.Client

	VirtualClient client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
}

type RegisterContext struct {
	Context          context.Context
	EventBroadcaster record.EventBroadcaster

	Options             *controllercontext.VirtualClusterOptions
	NodeServiceProvider nodeservice.NodeServiceProvider
	Controllers         map[string]bool
	LockFactory         locks.LockFactory

	TargetNamespace        string
	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	VirtualManager  ctrl.Manager
	PhysicalManager ctrl.Manager
}
