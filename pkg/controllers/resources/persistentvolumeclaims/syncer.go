package persistentvolumeclaims

import (
	"fmt"
	"time"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/pkg/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/snapshot"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
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

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.PersistentVolumeClaims())
	if err != nil {
		return nil, err
	}

	return &persistentVolumeClaimSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "persistent-volume-claim", &corev1.PersistentVolumeClaim{}, mapper),
		Importer:          pro.NewImporter(mapper),

		excludedAnnotations: []string{bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation},

		storageClassesEnabled:    ctx.Config.Sync.ToHost.StorageClasses.Enabled,
		schedulerEnabled:         ctx.Config.SchedulingInVirtualClusterEnabled(),
		useFakePersistentVolumes: !ctx.Config.Sync.ToHost.PersistentVolumes.Enabled,
	}, nil
}

type persistentVolumeClaimSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	excludedAnnotations []string

	storageClassesEnabled    bool
	schedulerEnabled         bool
	useFakePersistentVolumes bool
}

var _ syncertypes.OptionsProvider = &persistentVolumeClaimSyncer{}

func (s *persistentVolumeClaimSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		DisableUIDDeletion: true,
		ObjectCaching:      true,
	}
}

var _ syncertypes.Syncer = &persistentVolumeClaimSyncer{}

func (s *persistentVolumeClaimSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *persistentVolumeClaimSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.PersistentVolumeClaim]) (ctrl.Result, error) {
	// check if host PVC is currently being restored
	pObjName := s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.GetName(), Namespace: event.Virtual.GetNamespace()}, event.Virtual)
	restoreInProgress, err := s.isHostVolumeRestoreInProgress(ctx, pObjName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if host volume restore is in progress: %w", err)
	}
	if restoreInProgress {
		return ctrl.Result{
			RequeueAfter: 15 * time.Second,
		}, nil
	}

	if s.applyLimitByClass(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.HostOld, "host object was deleted", &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		s.EventRecorder().Event(event.Virtual, "Warning", "SyncError", err.Error())
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.PersistentVolumeClaims.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *persistentVolumeClaimSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.PersistentVolumeClaim]) (_ ctrl.Result, retErr error) {
	if s.applyLimitByClass(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	// check if host PVC is currently being restored
	hostObjName := types.NamespacedName{
		Name:      event.Host.GetName(),
		Namespace: event.Host.GetNamespace(),
	}
	restoreInProgress, err := s.isHostVolumeRestoreInProgress(ctx, hostObjName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if host volume restore is in progress: %w", err)
	}
	if restoreInProgress {
		return ctrl.Result{
			RequeueAfter: 15 * time.Second,
		}, nil
	}

	// if pvs are deleted check the corresponding pvc is deleted as well
	if event.Host.DeletionTimestamp != nil {
		if event.Virtual.DeletionTimestamp == nil {
			return patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.Host, "host persistent volume claim is being deleted", &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
		} else if *event.Virtual.DeletionGracePeriodSeconds != *event.Host.DeletionGracePeriodSeconds {
			return patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.Host, fmt.Sprintf("with grace period seconds %v", *event.Host.DeletionGracePeriodSeconds), &client.DeleteOptions{GracePeriodSeconds: event.Host.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(event.Virtual.UID))})
		}

		return ctrl.Result{}, nil
	} else if event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteHostObjectWithOptions(ctx, event.Host, event.Virtual, "virtual persistent volume claim is being deleted", &client.DeleteOptions{
			GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
		})
	}

	// make sure the persistent volume is synced / faked
	if event.Host.Spec.VolumeName != "" {
		requeue, err := s.ensurePersistentVolume(ctx, event.Host, event.Virtual, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.PersistentVolumeClaims.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}

		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// check backwards update
	s.translateUpdateBackwards(event.Host, event.Virtual)

	// copy host status
	event.Virtual.Status = *event.Host.Status.DeepCopy()

	// allow storage size to be increased
	event.Host.Spec.Resources.Requests = event.Virtual.Spec.Resources.Requests

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event, s.excludedAnnotations...)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *persistentVolumeClaimSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.PersistentVolumeClaim]) (_ ctrl.Result, retErr error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		// virtual object is not here anymore, so we delete
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vPvc := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host), s.excludedAnnotations...)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vPvc, event.Host, ctx.Config.Sync.ToHost.PersistentVolumeClaims.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vPvc, s.EventRecorder(), true)
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
			vName := mappings.HostToVirtual(ctx, pObj.Spec.VolumeName, "", nil, mappings.PersistentVolumes())
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

func (s *persistentVolumeClaimSyncer) isHostVolumeRestoreInProgress(ctx *synccontext.SyncContext, pObj types.NamespacedName) (bool, error) {
	configMaps := &corev1.ConfigMapList{}
	err := ctx.HostClient.List(ctx.Context, configMaps, client.InNamespace(ctx.Config.HostNamespace), client.MatchingLabels{
		constants.RestoreRequestLabel: "",
	})
	if err != nil {
		return false, err
	}

	pvcName := types.NamespacedName{
		Namespace: pObj.Namespace,
		Name:      pObj.Name,
	}.String()
	for _, configMap := range configMaps.Items {
		restoreRequest, err := snapshot.UnmarshalRestoreRequest(&configMap)
		if err != nil {
			return false, fmt.Errorf("unmarshal restore request: %w", err)
		}
		volumeRestore, ok := restoreRequest.Status.VolumesRestore.PersistentVolumeClaims[pvcName]
		if !ok {
			continue
		}
		if !(volumeRestore.CleaningUp() || volumeRestore.Done()) {
			return true, nil
		}
	}
	return false, nil
}

func recreatePersistentVolumeClaim(ctx *synccontext.SyncContext, virtualClient client.Client, vPV *corev1.PersistentVolume, vPVC *corev1.PersistentVolumeClaim, volumeName string, log loghelper.Logger) (*corev1.PersistentVolumeClaim, error) {
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

func (s *persistentVolumeClaimSyncer) applyLimitByClass(ctx *synccontext.SyncContext, virtual *corev1.PersistentVolumeClaim) bool {
	if !ctx.Config.Sync.FromHost.StorageClasses.Enabled.Bool() ||
		ctx.Config.Sync.FromHost.StorageClasses.Selector.Empty() ||
		virtual.Spec.StorageClassName == nil ||
		*virtual.Spec.StorageClassName == "" {
		return false
	}

	pStorageClass := &storagev1.StorageClass{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{Name: *virtual.Spec.StorageClassName}, pStorageClass)
	if err != nil || pStorageClass.GetDeletionTimestamp() != nil {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync persistent volume claim %q to host because the storage class %q couldn't be reached in the host: %s", virtual.GetName(), *virtual.Spec.StorageClassName, err)
		return true
	}
	matches, err := ctx.Config.Sync.FromHost.StorageClasses.Selector.Matches(pStorageClass)
	if err != nil {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync persistent volume claim %q to host because the storage class %q in the host could not be checked against the selector under 'sync.fromHost.storageClasses.selector': %s", virtual.GetName(), pStorageClass.GetName(), err)
		return true
	}
	if !matches {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync persistent volume claim %q to host because the storage class %q in the host does not match the selector under 'sync.fromHost.storageClasses.selector'", virtual.GetName(), pStorageClass.GetName())
		return true
	}

	return false
}
