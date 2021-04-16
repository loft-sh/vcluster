package persistentvolumeclaims

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
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

func Register(ctx *context2.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	return generic.RegisterSyncer(ctx, &syncer{
		useFakePersistentVolumes:     ctx.Options.UseFakePersistentVolumes,
		sharedPersistentVolumesMutex: ctx.LockFactory.GetLock("persistent-volumes-controller"),

		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "persistentvolumeclaim-syncer"}),
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
	}, "persistentvolumeclaim", generic.RegisterSyncerOptions{})
}

type syncer struct {
	useFakePersistentVolumes     bool
	sharedPersistentVolumesMutex sync.Locker

	eventRecoder    record.EventRecorder
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client
}

func (s *syncer) New() client.Object {
	return &corev1.PersistentVolumeClaim{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.PersistentVolumeClaimList{}
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	if vPvc.DeletionTimestamp != nil {
		// delete pvc immediately
		log.Debugf("delete virtual persistent volume claim %s/%s immediately, because it is being deleted & there is no physical persistent volume claim", vPvc.Namespace, vPvc.Name)
		err := s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	newObj, err := translate.SetupMetadata(s.targetNamespace, vPvc)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error setting metadata")
	}

	newPvc := newObj.(*corev1.PersistentVolumeClaim)
	log.Debugf("create physical persistent volume claim %s/%s", newPvc.Namespace, newPvc.Name)
	err = s.localClient.Create(ctx, newPvc)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vPvc.Namespace, vPvc.Name, err)
		s.eventRecoder.Eventf(vPvc, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	if vPvc.DeletionTimestamp != nil {
		if pPvc.DeletionTimestamp != nil {
			// pPod is under deletion, waiting for UWS bock populate the pod status.
			return ctrl.Result{}, nil
		}

		log.Debugf("delete physical persistent volume claim %s/%s, because virtual persistent volume claim is being deleted", pPvc.Namespace, pPvc.Name)
		err := s.localClient.Delete(ctx, pPvc, &client.DeleteOptions{
			GracePeriodSeconds: vPvc.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pPvc.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// update the virtual persistent volume claim if the spec has changed
	updatedPvc := calcPVCDiff(pPvc, vPvc)
	if updatedPvc != nil {
		log.Debugf("update physical persistent volume claim %s/%s, because spec or annotations have changed", updatedPvc.Namespace, updatedPvc.Name)
		err := s.localClient.Update(ctx, updatedPvc)
		if err != nil {
			s.eventRecoder.Eventf(vPvc, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	// update the virtual persistent volume claim if the spec has changed
	updatedPvc := calcPVCDiff(pPvc, vPvc)
	if updatedPvc != nil {
		return true, nil
	}

	return vPvc.DeletionTimestamp != nil && pPvc.DeletionTimestamp == nil, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	var err error
	if pPvc.DeletionTimestamp != nil {
		if vPvc.DeletionTimestamp == nil {
			log.Debugf("delete virtual persistent volume claim %s/%s, because the physical persistent volume claim is being deleted", vPvc.Namespace, vPvc.Name)
			if err = s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPvc.DeletionGracePeriodSeconds != *pPvc.DeletionGracePeriodSeconds {
			log.Debugf("delete virtual persistent volume claim %s/%s with grace period seconds %v", vPvc.Namespace, vPvc.Name, *pPvc.DeletionGracePeriodSeconds)
			if err = s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: pPvc.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPvc.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if pPvc.Spec.VolumeName != "" {
		err = s.ensurePersistentVolume(ctx, pPvc, vPvc, log)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	updated := calcPVCDiffBackwards(pPvc, vPvc)
	if updated != nil {
		log.Debugf("update virtual persistent volume claim %s/%s, because the spec has changed", vPvc.Namespace, vPvc.Name)
		err = s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		vPvc = updated
	}

	if !equality.Semantic.DeepEqual(vPvc.Status, pPvc.Status) {
		vPvc.Status = *pPvc.Status.DeepCopy()
		log.Debugf("update virtual persistent volume claim %s/%s, because the status has changed", vPvc.Namespace, vPvc.Name)
		err = s.virtualClient.Status().Update(ctx, vPvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ensurePersistentVolume(ctx context.Context, pObj *corev1.PersistentVolumeClaim, vObj *corev1.PersistentVolumeClaim, log loghelper.Logger) error {
	s.sharedPersistentVolumesMutex.Lock()
	defer s.sharedPersistentVolumesMutex.Unlock()

	// ensure the persistent volume is available in the virtual cluster
	vPV := &corev1.PersistentVolume{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.VolumeName}, vPV)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			log.Infof("error retrieving virtual pv %s: %v", pObj.Spec.VolumeName, err)
			return err
		}

		if s.useFakePersistentVolumes == true {
			// now insert it into the virtual cluster
			log.Debugf("create virtual fake pv %s, because pvc %s/%s uses it and it is not available in virtual cluster", pObj.Spec.VolumeName, vObj.Namespace, vObj.Name)

			// create fake persistent volume
			err = persistentvolumes.CreateFakePersistentVolume(ctx, s.virtualClient, types.NamespacedName{Name: pObj.Spec.VolumeName}, vObj)
			if err != nil {
				log.Infof("error creating virtual fake persistent volume %s: %v", pObj.Spec.VolumeName, err)
				return err
			}
		}
	}

	if vObj.Spec.VolumeName != pObj.Spec.VolumeName {
		log.Debugf("update virtual pvc %s/%s volume name to %s", vObj.Namespace, vObj.Name, pObj.Spec.VolumeName)

		vObj.Spec.VolumeName = pObj.Spec.VolumeName
		err = s.virtualClient.Update(ctx, vObj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	updated := calcPVCDiffBackwards(pPvc, vPvc)
	if updated != nil {
		return true, nil
	}

	return !equality.Semantic.DeepEqual(vPvc.Status, pPvc.Status), nil
}

func calcPVCDiff(pObj, vObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	var updated *corev1.PersistentVolumeClaim

	// allow storage size to be increased
	if pObj.Spec.Resources.Requests["storage"] != vObj.Spec.Resources.Requests["storage"] {
		updated = pObj.DeepCopy()
		if updated.Spec.Resources.Requests == nil {
			updated.Spec.Resources.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		updated.Spec.Resources.Requests["storage"] = vObj.Spec.Resources.Requests["storage"]
	}

	if !translate.EqualExcept(pObj.Annotations, vObj.Annotations, bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Annotations = translate.SetExcept(vObj.Annotations, updated.Annotations, bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation)
	}

	return updated
}

func calcPVCDiffBackwards(pObj, vObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	var updated *corev1.PersistentVolumeClaim

	// check for metadata annotations
	if pObj.Annotations != nil && (vObj.Annotations == nil ||
		vObj.Annotations[bindCompletedAnnotation] != pObj.Annotations[bindCompletedAnnotation] ||
		vObj.Annotations[boundByControllerAnnotation] != pObj.Annotations[boundByControllerAnnotation] ||
		vObj.Annotations[storageProvisionerAnnotation] != pObj.Annotations[storageProvisionerAnnotation]) {
		updated = vObj.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}

		updated.Annotations[bindCompletedAnnotation] = pObj.Annotations[bindCompletedAnnotation]
		updated.Annotations[boundByControllerAnnotation] = pObj.Annotations[boundByControllerAnnotation]
		updated.Annotations[storageProvisionerAnnotation] = pObj.Annotations[storageProvisionerAnnotation]
	}

	return updated
}
