package endpoints

import (
	"context"
	"encoding/json"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	if ctx.Options.SyncServiceSelector {
		return NewKubernetesEndpointsSyncer(ctx)
	}

	return &endpointsSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "endpoints", &corev1.Endpoints{}),
		serviceName:          ctx.Options.ServiceName,
	}, nil
}

type endpointsSyncer struct {
	translator.NamespacedTranslator

	serviceName string
}

func (s *endpointsSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(ctx, vObj))
}

func (s *endpointsSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownUpdate(ctx, vObj, s.translateUpdate(ctx, pObj.(*corev1.Endpoints), vObj.(*corev1.Endpoints)))
}

var _ syncer.Starter = &endpointsSyncer{}

func (s *endpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	// don't do anything for the kubernetes service
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesServiceEndpoints(ctx.Context, ctx.VirtualClient, ctx.CurrentNamespaceClient, ctx.CurrentNamespace, s.serviceName)
	}

	return false, nil
}

func (s *endpointsSyncer) ReconcileEnd() {}

func SyncKubernetesServiceEndpoints(ctx context.Context, virtualClient client.Client, localClient client.Client, serviceNamespace, serviceName string) error {
	// get physical service endpoints
	pObj := &corev1.Endpoints{}
	err := localClient.Get(ctx, types.NamespacedName{
		Namespace: serviceNamespace,
		Name:      serviceName,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// get virtual service endpoints
	vObj := &corev1.Endpoints{}
	err = virtualClient.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      "kubernetes",
	}, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// build new subsets
	newSubsets := pObj.DeepCopy().Subsets
	for i := range newSubsets {
		for j := range newSubsets[i].Ports {
			newSubsets[i].Ports[j].Name = "https"
		}
		for j := range pObj.Subsets[i].Addresses {
			newSubsets[i].Addresses[j].Hostname = ""
			newSubsets[i].Addresses[j].NodeName = nil
			newSubsets[i].Addresses[j].TargetRef = nil
		}
		for j := range pObj.Subsets[i].NotReadyAddresses {
			newSubsets[i].NotReadyAddresses[j].Hostname = ""
			newSubsets[i].NotReadyAddresses[j].NodeName = nil
			newSubsets[i].NotReadyAddresses[j].TargetRef = nil
		}
	}

	oldJSON, err := json.Marshal(vObj.Subsets)
	if err != nil {
		return err
	}
	newJSON, err := json.Marshal(newSubsets)
	if err != nil {
		return err
	}

	if string(oldJSON) == string(newJSON) {
		return nil
	}

	vObj.Subsets = newSubsets
	return virtualClient.Update(ctx, vObj)
}
