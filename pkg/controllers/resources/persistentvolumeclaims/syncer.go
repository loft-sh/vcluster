package persistentvolumeclaims

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/pkg/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
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
	storageClassesEnabled := ctx.Config.Sync.ToHost.StorageClasses.Enabled
	excludedAnnotations := []string{bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation}
	return &persistentVolumeClaimSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "persistent-volume-claim", &corev1.PersistentVolumeClaim{}, mappings.PersistentVolumeClaims(), excludedAnnotations...),

		storageClassesEnabled:    storageClassesEnabled,
		schedulerEnabled:         ctx.Config.ControlPlane.Advanced.VirtualScheduler.Enabled,
		useFakePersistentVolumes: !ctx.Config.Sync.ToHost.PersistentVolumes.Enabled,
	}, nil
}

type persistentVolumeClaimSyncer struct {
	syncer.GenericTranslator

	storageClassesEnabled    bool
	schedulerEnabled         bool
	useFakePersistentVolumes bool
}

var _ syncer.OptionsProvider = &persistentVolumeClaimSyncer{}

func (s *persistentVolumeClaimSyncer) Options() *syncer.Options {
	return &syncer.Options{
		DisableUIDDeletion: true,
	}
}

var _ syncer.Syncer = &persistentVolumeClaimSyncer{}

func (s *persistentVolumeClaimSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	if ctx.IsDelete || vPvc.DeletionTimestamp != nil {
		// delete pvc immediately
		ctx.Log.Infof("delete virtual persistent volume claim %s/%s immediately, because it is being deleted & there is no physical persistent volume claim", vPvc.Namespace, vPvc.Name)
		err := ctx.VirtualClient.Delete(ctx, vPvc, &client.DeleteOptions{
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

	return s.SyncToHostCreate(ctx, vObj, newPvc)
}

func (s *persistentVolumeClaimSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	// if pvs are deleted check the corresponding pvc is deleted as well
	if pPvc.DeletionTimestamp != nil {
		if vPvc.DeletionTimestamp == nil {
			ctx.Log.Infof("delete virtual persistent volume claim %s/%s, because the physical persistent volume claim is being deleted", vPvc.Namespace, vPvc.Name)
			if err := ctx.VirtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPvc.DeletionGracePeriodSeconds != *pPvc.DeletionGracePeriodSeconds {
			ctx.Log.Infof("delete virtual persistent volume claim %s/%s with grace period seconds %v", vPvc.Namespace, vPvc.Name, *pPvc.DeletionGracePeriodSeconds)
			if err := ctx.VirtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: pPvc.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPvc.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if vPvc.DeletionTimestamp != nil {
		ctx.Log.Infof("delete physical persistent volume claim %s/%s, because virtual persistent volume claim is being deleted", pPvc.Namespace, pPvc.Name)
		err := ctx.PhysicalClient.Delete(ctx, pPvc, &client.DeleteOptions{
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
		requeue, err := s.ensurePersistentVolume(ctx, pPvc, vPvc, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, pPvc, vPvc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pPvc, vPvc); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}

		if retErr != nil {
			s.EventRecorder().Eventf(vPvc, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// check backwards update
	s.translateUpdateBackwards(pPvc, vPvc)

	// copy host status
	vPvc.Status = *pPvc.Status.DeepCopy()

	// forward update
	s.translateUpdate(ctx, pPvc, vPvc)

	return ctrl.Result{}, nil
}

func (s *persistentVolumeClaimSyncer) ensurePersistentVolume(ctx *synccontext.SyncContext, pObj *corev1.PersistentVolumeClaim, vObj *corev1.PersistentVolumeClaim, log loghelper.Logger) (bool, error) {
	// ensure the persistent volume is available in the virtual cluster
	vPV := &corev1.PersistentVolume{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.VolumeName}, vPV)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			log.Infof("error retrieving virtual pv %s: %v", pObj.Spec.VolumeName, err)
			return false, err
		}
	}

	if pObj.Spec.VolumeName != "" && vObj.Spec.VolumeName != pObj.Spec.VolumeName {
		newVolumeName := pObj.Spec.VolumeName
		if !s.useFakePersistentVolumes {
			vName := mappings.PersistentVolumes().HostToVirtual(ctx, types.NamespacedName{Name: pObj.Spec.VolumeName}, nil)
			if vName.Name == "" {
				log.Infof("error retrieving virtual persistent volume %s: not found", pObj.Spec.VolumeName)
				return false, fmt.Errorf("error retrieving virtual persistent volume %s: not found", pObj.Spec.VolumeName)
			}

			newVolumeName = vName.Name
		}

		if newVolumeName != vObj.Spec.VolumeName {
			if vObj.Spec.VolumeName != "" {
				log.Infof("recreate persistent volume claim because volumeName differs between physical and virtual pvc: %s != %s", vObj.Spec.VolumeName, newVolumeName)
				s.EventRecorder().Eventf(vObj, corev1.EventTypeWarning, "VolumeNameDiffers", "recreate persistent volume claim because volumeName differs between physical and virtual pvc: %s != %s", vObj.Spec.VolumeName, newVolumeName)
				_, err = recreatePersistentVolumeClaim(ctx, ctx.VirtualClient, vPV, vObj, newVolumeName, log)
				if err != nil {
					log.Infof("error recreating virtual persistent volume claim: %v", err)
					return false, err
				}

				return true, nil
			}

			log.Infof("update virtual pvc %s/%s volume name to %s", vObj.Namespace, vObj.Name, newVolumeName)
			vObj.Spec.VolumeName = newVolumeName
			err = ctx.VirtualClient.Update(ctx, vObj)
			if err != nil {
				return false, err
			}
		}
	}

	return false, nil
}

func recreatePersistentVolumeClaim(ctx context.Context, virtualClient client.Client, vPV *corev1.PersistentVolume, vPVC *corev1.PersistentVolumeClaim, volumeName string, log loghelper.Logger) (*corev1.PersistentVolumeClaim, error) {
	// check if we should lock the pv from deletion
	if vPV != nil && vPV.Name != "" {
		// lock pv
		before := vPV.DeepCopy()
		if vPV.Annotations == nil {
			vPV.Annotations = map[string]string{}
		}
		timestamp, err := metav1.Now().MarshalText()
		if err != nil {
			return nil, errors.Wrap(err, "marshal time")
		}
		vPV.Annotations[persistentvolumes.LockPersistentVolume] = string(timestamp)
		err = virtualClient.Patch(ctx, vPV, client.MergeFrom(before))
		if err != nil {
			return nil, errors.Wrap(err, "patch persistent volume")
		}

		// reset pv
		defer func() {
			before := vPV.DeepCopy()
			delete(vPV.Annotations, persistentvolumes.LockPersistentVolume)

			err := virtualClient.Patch(ctx, vPV, client.MergeFrom(before))
			if err != nil {
				log.Errorf("error resetting pv %s: %v", vPV.Name, err)
			}
		}()
	}

	// remove finalizers & delete
	if len(vPVC.Finalizers) > 0 {
		vPVC.Finalizers = []string{}
		err := virtualClient.Update(ctx, vPVC)
		if err != nil {
			return nil, errors.Wrap(err, "remove finalizers")
		}
	}

	// delete & create with correct volume name
	err := virtualClient.Delete(ctx, vPVC)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "delete pvc")
	}

	// make sure we don't set the resource version during create
	vPVC = vPVC.DeepCopy()
	vPVC.ResourceVersion = ""
	vPVC.UID = ""
	vPVC.DeletionTimestamp = nil
	vPVC.Generation = 0
	vPVC.Spec.VolumeName = volumeName

	// create the new service with the correct volume name
	err = virtualClient.Create(ctx, vPVC)
	if err != nil {
		klog.Errorf("error recreating virtual pvc: %s/%s: %v", vPVC.Namespace, vPVC.Name, err)
		return nil, errors.Wrap(err, "create pvc")
	}

	return vPVC, nil
}
