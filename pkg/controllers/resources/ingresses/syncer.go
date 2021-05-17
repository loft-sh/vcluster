package ingresses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

func Register(ctx *context2.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	return generic.RegisterSyncer(ctx, &syncer{
		sharedMutex:     ctx.LockFactory.GetLock("ingress-controller"),
		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "ingress-syncer"}),
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
	}, "ingress", generic.RegisterSyncerOptions{})
}

type syncer struct {
	sharedMutex     sync.Locker
	eventRecoder    record.EventRecorder
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client
}

func (s *syncer) New() client.Object {
	return &networkingv1beta1.Ingress{}
}

func (s *syncer) NewList() client.ObjectList {
	return &networkingv1beta1.IngressList{}
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vIngress := vObj.(*networkingv1beta1.Ingress)
	newObj, err := translate.SetupMetadata(s.targetNamespace, vIngress)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error setting metadata")
	}

	newIngress := newObj.(*networkingv1beta1.Ingress)
	newIngress.Spec = *translateSpec(vIngress.Namespace, &vIngress.Spec)
	log.Debugf("create physical ingress %s/%s", newIngress.Namespace, newIngress.Name)
	err = s.localClient.Create(ctx, newIngress)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vIngress.Namespace, vIngress.Name, err)
		s.eventRecoder.Eventf(vIngress, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	var err error
	vIngress := vObj.(*networkingv1beta1.Ingress)
	pIngress := pObj.(*networkingv1beta1.Ingress)

	// did something change?
	updateNeeded, _ := s.ForwardUpdateNeeded(pIngress, vIngress)
	if updateNeeded {
		pIngress = pIngress.DeepCopy()
		pIngress.Annotations = vIngress.Annotations
		pIngress.Labels = translate.TranslateLabels(vIngress.Namespace, vIngress.Labels)
		pIngress.Spec = *translateSpec(vIngress.Namespace, &vIngress.Spec)
		log.Debugf("updating physical ingress %s/%s, because virtual ingress spec or annotations have changed", pIngress.Namespace, pIngress.Name)
		err = s.localClient.Update(ctx, pIngress)
		if err != nil {
			s.eventRecoder.Eventf(vIngress, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vIngress := vObj.(*networkingv1beta1.Ingress)
	pIngress := pObj.(*networkingv1beta1.Ingress)

	return !equality.Semantic.DeepEqual(*translateSpec(vIngress.Namespace, &vIngress.Spec), pIngress.Spec) ||
		!equality.Semantic.DeepEqual(vIngress.Annotations, pIngress.Annotations) ||
		!translate.LabelsEqual(vIngress.Namespace, vIngress.Labels, pIngress.Labels), nil
}

func translateSpec(namespace string, vIngressSpec *networkingv1beta1.IngressSpec) *networkingv1beta1.IngressSpec {
	retSpec := vIngressSpec.DeepCopy()
	if retSpec.Backend != nil {
		if retSpec.Backend.ServiceName != "" {
			retSpec.Backend.ServiceName = translate.PhysicalName(retSpec.Backend.ServiceName, namespace)
		}
		if retSpec.Backend.Resource != nil {
			retSpec.Backend.Resource.Name = translate.PhysicalName(retSpec.Backend.Resource.Name, namespace)
		}
	}

	for i, rule := range retSpec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName != "" {
					retSpec.Rules[i].HTTP.Paths[j].Backend.ServiceName = translate.PhysicalName(retSpec.Rules[i].HTTP.Paths[j].Backend.ServiceName, namespace)
				}
				if path.Backend.Resource != nil {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name = translate.PhysicalName(retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name, namespace)
				}
			}
		}
	}

	for i, tls := range retSpec.TLS {
		if tls.SecretName != "" {
			retSpec.TLS[i].SecretName = translate.PhysicalName(retSpec.TLS[i].SecretName, namespace)
		}
	}

	return retSpec
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vIngress := vObj.(*networkingv1beta1.Ingress)
	pIngress := pObj.(*networkingv1beta1.Ingress)

	var err error
	if !equality.Semantic.DeepEqual(vIngress.Spec.IngressClassName, pIngress.Spec.IngressClassName) {
		newIngress := vIngress.DeepCopy()
		newIngress.Spec.IngressClassName = pIngress.Spec.IngressClassName

		log.Debugf("update virtual ingress %s/%s, because ingress class name is out of sync", vIngress.Namespace, vIngress.Name)
		err = s.virtualClient.Update(ctx, newIngress)
		if err != nil {
			return ctrl.Result{}, err
		}

		vIngress = newIngress
	}

	if !equality.Semantic.DeepEqual(vIngress.Status, pIngress.Status) {
		newIngress := vIngress.DeepCopy()
		newIngress.Status = pIngress.Status
		log.Debugf("update virtual ingress %s/%s, because status is out of sync", vIngress.Namespace, vIngress.Name)
		err = s.virtualClient.Status().Update(ctx, newIngress)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vIngress := vObj.(*networkingv1beta1.Ingress)
	pIngress := pObj.(*networkingv1beta1.Ingress)

	return !equality.Semantic.DeepEqual(vIngress.Spec.IngressClassName, pIngress.Spec.IngressClassName) || !equality.Semantic.DeepEqual(vIngress.Status, pIngress.Status), nil
}

func (s *syncer) BackwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.sharedMutex.Lock()

	return false, nil
}

func (s *syncer) BackwardEnd() {
	s.sharedMutex.Unlock()
}

func (s *syncer) ForwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.sharedMutex.Lock()

	return false, nil
}

func (s *syncer) ForwardEnd() {
	s.sharedMutex.Unlock()
}
