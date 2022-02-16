package persistentvolumeclaims

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

const (
	bindCompletedAnnotation      = "pv.kubernetes.io/bind-completed"
	boundByControllerAnnotation  = "pv.kubernetes.io/bound-by-controller"
	storageProvisionerAnnotation = "volume.beta.kubernetes.io/storage-provisioner"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &persistentVolumeClaimSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "persistent-volume-claim", &corev1.PersistentVolumeClaim{}, bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation),

		storageClassesEnabled:    ctx.Controllers["storageclasses"],
		useFakePersistentVolumes: !ctx.Controllers["persistentvolumes"],
	}, nil
}

type persistentVolumeClaimSyncer struct {
	translator.NamespacedTranslator

	storageClassesEnabled    bool
	useFakePersistentVolumes bool
}

var _ syncer.Syncer = &persistentVolumeClaimSyncer{}

func (s *persistentVolumeClaimSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	if vPvc.DeletionTimestamp != nil {
		// delete pvc immediately
		ctx.Log.Infof("delete virtual persistent volume claim %s/%s immediately, because it is being deleted & there is no physical persistent volume claim", vPvc.Namespace, vPvc.Name)
		err := ctx.VirtualClient.Delete(ctx.Context, vPvc, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	newPvc, err := s.translate(ctx, vPvc)
	if err != nil {
		s.EventRecorder().Event(vPvc, "Warning", "SyncError", err.Error())
		return ctrl.Result{}, err
	}

	return s.SyncDownCreate(ctx, vObj, newPvc)
}

func (s *persistentVolumeClaimSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	// if pvs are deleted check the corresponding pvc is deleted as well
	if pPvc.DeletionTimestamp != nil {
		if vPvc.DeletionTimestamp == nil {
			ctx.Log.Infof("delete virtual persistent volume claim %s/%s, because the physical persistent volume claim is being deleted", vPvc.Namespace, vPvc.Name)
			if err := ctx.VirtualClient.Delete(ctx.Context, vPvc, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPvc.DeletionGracePeriodSeconds != *pPvc.DeletionGracePeriodSeconds {
			ctx.Log.Infof("delete virtual persistent volume claim %s/%s with grace period seconds %v", vPvc.Namespace, vPvc.Name, *pPvc.DeletionGracePeriodSeconds)
			if err := ctx.VirtualClient.Delete(ctx.Context, vPvc, &client.DeleteOptions{GracePeriodSeconds: pPvc.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPvc.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if vPvc.DeletionTimestamp != nil {
		ctx.Log.Infof("delete physical persistent volume claim %s/%s, because virtual persistent volume claim is being deleted", pPvc.Namespace, pPvc.Name)
		err := ctx.PhysicalClient.Delete(ctx.Context, pPvc, &client.DeleteOptions{
			GracePeriodSeconds: vPvc.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pPvc.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// make sure the persistent volume is synced / faked
	if pPvc.Spec.VolumeName != "" {
		err := s.ensurePersistentVolume(ctx, pPvc, vPvc, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// check backwards update
	updated := s.translateUpdateBackwards(pPvc, vPvc)
	if updated != nil {
		ctx.Log.Infof("update virtual persistent volume claim %s/%s, because the spec has changed", vPvc.Namespace, vPvc.Name)
		err := ctx.VirtualClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	// check backwards status
	if !equality.Semantic.DeepEqual(vPvc.Status, pPvc.Status) {
		vPvc.Status = *pPvc.Status.DeepCopy()
		ctx.Log.Infof("update virtual persistent volume claim %s/%s, because the status has changed", vPvc.Namespace, vPvc.Name)
		err := ctx.VirtualClient.Status().Update(ctx.Context, vPvc)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	// forward update
	return s.SyncDownUpdate(ctx, vPvc, s.translateUpdate(pPvc, vPvc))
}

func (s *persistentVolumeClaimSyncer) ensurePersistentVolume(ctx *synccontext.SyncContext, pObj *corev1.PersistentVolumeClaim, vObj *corev1.PersistentVolumeClaim, log loghelper.Logger) error {
	// ensure the persistent volume is available in the virtual cluster
	vPV := &corev1.PersistentVolume{}
	err := ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{Name: pObj.Spec.VolumeName}, vPV)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			log.Infof("error retrieving virtual pv %s: %v", pObj.Spec.VolumeName, err)
			return err
		}
	}

	if pObj.Spec.VolumeName != "" && vObj.Spec.VolumeName != pObj.Spec.VolumeName {
		newVolumeName := pObj.Spec.VolumeName
		if !s.useFakePersistentVolumes {
			vObj := &corev1.PersistentVolume{}
			err = clienthelper.GetByIndex(ctx.Context, ctx.VirtualClient, vObj, constants.IndexByPhysicalName, pObj.Spec.VolumeName)
			if err != nil {
				log.Infof("error retrieving virtual persistent volume %s: %v", pObj.Spec.VolumeName, err)
				return err
			}

			newVolumeName = vObj.Name
		}

		if newVolumeName != vObj.Spec.VolumeName {
			log.Infof("update virtual pvc %s/%s volume name to %s", vObj.Namespace, vObj.Name, newVolumeName)

			vObj.Spec.VolumeName = newVolumeName
			err = ctx.VirtualClient.Update(ctx.Context, vObj)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
