package persistentvolumes

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolume{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translatePersistentVolumeName(ctx.Options.TargetNamespace, rawObj.(*corev1.PersistentVolume).Name, rawObj)}
	})
	if err != nil {
		return err
	}

	return generic.RegisterSyncer(ctx, "persistentvolume", &syncer{
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),

		metadataTranslator: translator.NewClusterTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.PersistentVolume{}, NewPersistentVolumeTranslator(ctx.Options.TargetNamespace), HostClusterPersistentVolumeAnnotation),
	})
}

func mapPVCs(obj client.Object) []reconcile.Request {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return nil
	}

	if pvc.Spec.VolumeName != "" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pvc.Spec.VolumeName,
				},
			},
		}
	}

	return nil
}

func NewPersistentVolumeTranslator(physicalNamespace string) translator.PhysicalNameTranslator {
	return func(vName string, vObj client.Object) string {
		return translatePersistentVolumeName(physicalNamespace, vName, vObj)
	}
}

type syncer struct {
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client

	metadataTranslator translator.MetadataTranslator
}

func (s *syncer) New() client.Object {
	return &corev1.PersistentVolume{}
}

var _ generic.ControllerModifier = &syncer{}

func (s *syncer) ModifyController(builder *builder.Builder) *builder.Builder {
	return builder.Watches(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, handler.EnqueueRequestsFromMapFunc(mapPVCs))
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPv := vObj.(*corev1.PersistentVolume)
	if vPv.DeletionTimestamp != nil || (vPv.Annotations != nil && vPv.Annotations[HostClusterPersistentVolumeAnnotation] != "") {
		if len(vPv.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vPv.Finalizers = []string{}
			log.Infof("remove virtual persistent volume %s finalizers, because object should get deleted", vPv.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx, vPv)
		}

		log.Infof("remove virtual persistent volume %s, because object should get deleted", vPv.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vPv)
	}

	pPv := s.translate(vPv)
	log.Infof("create physical persistent volume %s, because there is a virtual persistent volume", pPv.Name)
	err := s.localClient.Create(ctx, pPv)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)

	// check if objects are getting deleted
	if vObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			log.Infof("delete physical persistent volume %s, because virtual persistent volume is terminating", pObj.GetName())
			err := s.localClient.Delete(ctx, pObj)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the persistent volume should get synced
	sync, vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		log.Infof("delete virtual persistent volume %s, because there is no virtual persistent volume claim with that volume", vPersistentVolume.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
	}

	// check if there is a corresponding virtual pvc
	updatedObj := s.translateUpdateBackwards(vPersistentVolume, pPersistentVolume, vPvc)
	if updatedObj != nil {
		log.Infof("update virtual persistent volume %s, because spec has changed", vPersistentVolume.Name)
		err = s.virtualClient.Update(ctx, updatedObj)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will reconcile anyways
		return ctrl.Result{}, nil
	}

	// check status
	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, pPersistentVolume.Status) {
		vPersistentVolume.Status = *pPersistentVolume.Status.DeepCopy()
		log.Infof("update virtual persistent volume %s, because status has changed", vPersistentVolume.Name)
		err = s.virtualClient.Status().Update(ctx, vPersistentVolume)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will reconcile anyways
		return ctrl.Result{}, nil
	}

	// update the physical persistent volume if the virtual has changed
	if vPersistentVolume.Annotations == nil || vPersistentVolume.Annotations[HostClusterPersistentVolumeAnnotation] == "" {
		if vPersistentVolume.DeletionTimestamp != nil {
			if pPersistentVolume.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}

			log.Infof("delete physical persistent volume %s, because virtual persistent volume is being deleted", pPersistentVolume.Name)
			err := s.localClient.Delete(ctx, pPersistentVolume, &client.DeleteOptions{
				GracePeriodSeconds: vPersistentVolume.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pPersistentVolume.UID)),
			})
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		updatedPv := s.translateUpdate(vPersistentVolume, pPersistentVolume)
		if updatedPv != nil {
			log.Infof("update physical persistent volume %s, because spec or annotations have changed", updatedPv.Name)
			err := s.localClient.Update(ctx, updatedPv)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

var _ generic.BackwardSyncer = &syncer{}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	sync, vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if translate.IsManagedCluster(s.targetNamespace, pObj) {
		log.Infof("delete physical persistent volume %s, because it is not needed anymore", pPersistentVolume.Name)
		return generic.DeleteObject(ctx, s.localClient, pObj, log)
	} else if sync {
		// create the persistent volume
		vObj := s.translateBackwards(pPersistentVolume, vPvc)
		if vPvc != nil {
			log.Infof("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
		}

		return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
	}

	return ctrl.Result{}, nil
}

func (s *syncer) shouldSync(ctx context.Context, pObj *corev1.PersistentVolume) (bool, *corev1.PersistentVolumeClaim, error) {
	// is there an assigned PVC?
	if pObj.Spec.ClaimRef == nil || pObj.Spec.ClaimRef.Namespace != s.targetNamespace {
		if translate.IsManagedCluster(s.targetNamespace, pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	vPvc := &corev1.PersistentVolumeClaim{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vPvc, constants.IndexByPhysicalName, pObj.Spec.ClaimRef.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.IsManagedCluster(s.targetNamespace, pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	return true, vPvc, nil
}

func (s *syncer) IsManaged(pObj client.Object) (bool, error) {
	pPv, ok := pObj.(*corev1.PersistentVolume)
	if !ok {
		return false, nil
	}

	sync, _, err := s.shouldSync(context.TODO(), pPv)
	if err != nil {
		return false, nil
	}

	return sync, nil
}

func (s *syncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translatePersistentVolumeName(s.targetNamespace, req.Name, vObj)}
}

func (s *syncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[translator.NameAnnotation] != "" {
		return types.NamespacedName{
			Name: pAnnotations[translator.NameAnnotation],
		}
	}

	vObj := &corev1.PersistentVolume{}
	err := clienthelper.GetByIndex(context.Background(), s.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetName())
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: pObj.GetName()}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translatePersistentVolumeName(physicalNamespace, name string, vObj runtime.Object) string {
	if vObj == nil {
		return name
	}

	vPv, ok := vObj.(*corev1.PersistentVolume)
	if !ok || vPv.Annotations == nil || vPv.Annotations[HostClusterPersistentVolumeAnnotation] == "" {
		return translate.PhysicalNameClusterScoped(name, physicalNamespace)
	}

	return vPv.Annotations[HostClusterPersistentVolumeAnnotation]
}
