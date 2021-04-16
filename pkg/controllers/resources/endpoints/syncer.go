package endpoints

import (
	"context"
	"encoding/json"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func Register(ctx *context2.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	return generic.RegisterSyncer(ctx, &syncer{
		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "endpoints-syncer"}),
		targetNamespace: ctx.Options.TargetNamespace,
		serviceName:     ctx.Options.ServiceName,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
	}, "endpoints", generic.RegisterSyncerOptions{})
}

type syncer struct {
	eventRecoder    record.EventRecorder
	targetNamespace string
	serviceName     string
	localClient     client.Client
	virtualClient   client.Client
}

func (s *syncer) New() client.Object {
	return &corev1.Endpoints{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.EndpointsList{}
}

func (s *syncer) translate(vObj runtime.Object) (*corev1.Endpoints, error) {
	newObj, err := translate.SetupMetadata(s.targetNamespace, vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	return newObj.(*corev1.Endpoints), nil
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vEndpoints := vObj.(*corev1.Endpoints)
	newEndpoints, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = clienthelper.Apply(ctx, s.localClient, newEndpoints, log)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vEndpoints.Namespace, vEndpoints.Name, err)
		s.eventRecoder.Eventf(vEndpoints, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardCreateNeeded(vObj client.Object) (bool, error) {
	// dont do anything for the kubernetes endpoints
	vEndpoints := vObj.(*corev1.Endpoints)
	if vEndpoints.Name == "kubernetes" && vEndpoints.Namespace == "default" {
		return false, nil
	}

	return true, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	return s.ForwardCreate(ctx, vObj, log)
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	newSecret, err := s.translate(vObj)
	if err != nil {
		return false, err
	}

	equal, err := clienthelper.AppliedObjectsEqual(pObj, newSecret)
	if err != nil {
		return false, err
	}

	return equal == false, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	return false, nil
}

func (s *syncer) BackwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	// sync the kubernetes service
	if req.Name == s.serviceName && req.Namespace == s.targetNamespace {
		return true, SyncKubernetesServiceEndpoints(ctx, s.virtualClient, s.localClient, s.targetNamespace, s.serviceName)
	}

	return false, nil
}

func (s *syncer) BackwardEnd() {

}

func (s *syncer) ForwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	// dont do anything for the kubernetes service
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return true, SyncKubernetesServiceEndpoints(ctx, s.virtualClient, s.localClient, s.targetNamespace, s.serviceName)
	}

	return false, nil
}

func (s *syncer) ForwardEnd() {

}

func SyncKubernetesServiceEndpoints(ctx context.Context, virtualClient client.Client, localClient client.Client, targetNamespace, serviceName string) error {
	// get physical service endpoints
	pObj := &corev1.Endpoints{}
	err := localClient.Get(ctx, types.NamespacedName{
		Namespace: targetNamespace,
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
		newSubsets[i].NotReadyAddresses = nil
		for j := range newSubsets[i].Ports {
			newSubsets[i].Ports[j].Name = "https"
		}
		for j := range pObj.Subsets[i].Addresses {
			newSubsets[i].Addresses[j].Hostname = ""
			newSubsets[i].Addresses[j].NodeName = nil
			newSubsets[i].Addresses[j].TargetRef = nil
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
