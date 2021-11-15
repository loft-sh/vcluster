package endpoints

import (
	"context"
	"encoding/json"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	return generic.RegisterSyncerIndices(ctx, &corev1.Endpoints{})
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	var (
		err error
		serviceClient = ctx.LocalManager.GetClient()
	)
	if ctx.Options.ServiceNamespace != ctx.Options.TargetNamespace {
		serviceClient, err = client.New(ctx.LocalManager.GetConfig(), client.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			Mapper: ctx.LocalManager.GetRESTMapper(),
		})
		if err != nil {
			return errors.Wrap(err, "create uncached client")
		}
	}

	return generic.RegisterSyncer(ctx, "endpoints", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.Endpoints{}),

		targetNamespace:  ctx.Options.TargetNamespace,
		serviceName:      ctx.Options.ServiceName,
		serviceNamespace: ctx.Options.ServiceNamespace,
		serviceClient:    serviceClient,
		virtualClient:    ctx.VirtualManager.GetClient(),
		
		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "endpoints-syncer"}), "endpoints"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace),
	})
}

type syncer struct {
	generic.Translator
	targetNamespace string

	serviceName      string
	serviceNamespace string
	serviceClient    client.Client

	virtualClient client.Client

	creator *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &corev1.Endpoints{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pObj, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	updated, err := s.translateUpdate(pObj.(*corev1.Endpoints), vObj.(*corev1.Endpoints))
	if err != nil {
		return ctrl.Result{}, err
	}
	
	return s.creator.Update(ctx, vObj, updated, log)
}

var _ generic.Starter = &syncer{}

func (s *syncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	// dont do anything for the kubernetes service
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesServiceEndpoints(ctx, s.virtualClient, s.serviceClient, s.serviceNamespace, s.serviceName)
	}

	return false, nil
}

func (s *syncer) ReconcileEnd() {}

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
