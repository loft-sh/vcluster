package endpoints

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubernetesEndpointsSyncer(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &kubernetesEndpointsSyncer{
		serviceName: ctx.Options.ServiceName,
	}, nil
}

type kubernetesEndpointsSyncer struct {
	serviceName string
}

func (r *kubernetesEndpointsSyncer) Resource() client.Object {
	return &corev1.Endpoints{}
}

func (r *kubernetesEndpointsSyncer) Name() string {
	return "kubernetes-endpoints-syncer"
}

var _ syncer.FakeSyncer = &kubernetesEndpointsSyncer{}

func (r *kubernetesEndpointsSyncer) FakeSyncUp(ctx *synccontext.SyncContext, name types.NamespacedName) (ctrl.Result, error) {
	return ctrl.Result{}, fmt.Errorf("unexpected sync")
}

func (r *kubernetesEndpointsSyncer) FakeSync(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return ctrl.Result{}, fmt.Errorf("unexpected sync")
}

var _ syncer.Starter = &kubernetesEndpointsSyncer{}

func (r *kubernetesEndpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesServiceEndpoints(ctx.Context, ctx.VirtualClient, ctx.CurrentNamespaceClient, ctx.CurrentNamespace, r.serviceName)
	}

	return true, nil
}

func (r *kubernetesEndpointsSyncer) ReconcileEnd() {}
