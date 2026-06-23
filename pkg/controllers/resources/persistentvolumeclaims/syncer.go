package persistentvolumeclaims

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
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
	selectedNodeAnnotation       = "volume.kubernetes.io/selected-node"

	dataProtectionAPIGroup               = "dataprotection.kubeblocks.io"
	dataProtectionBackupKind             = "Backup"
	dataProtectionPopulateFromAnnotation = "dataprotection.kubeblocks.io/populate-from"

	dataProtectionMaterializationRequestLabel  = "vcluster.loft.sh/dataprotection-materialization-request"
	dataProtectionMaterializationRequestPrefix = "dp-host-materialization-"
	dataProtectionMaterializationStatePending  = "pending"

	dataProtectionRestoreConditionReasonProvisioned = "Provisioned"
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

	preserveDeletingHostPVC, err := s.shouldPreserveDataProtectionNoDataRestorePVCWhileHostDeleting(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}
	if event.HostOld != nil && preserveDeletingHostPVC && event.Virtual.DeletionTimestamp == nil {
		// The host PVC was intentionally deleted so it can be recreated without
		// the Backup dataSource. Keep the virtual restore PVC and continue into
		// the create path below.
	} else if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.HostOld, "host object was deleted", &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
	}

	pObj, handled, err := s.translateDataProtectionBackupToHost(ctx, event.Virtual)
	if err != nil {
		s.EventRecorder().Eventf(
			event.Virtual,
			nil,
			"Warning",
			"SyncError",
			fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
			err.Error(),
		)
		return ctrl.Result{}, err
	}
	if handled {
		err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.PersistentVolumeClaims.Patches, false)
		if err != nil {
			return ctrl.Result{}, err
		}

		return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
	}

	pObj, err = s.translate(ctx, event.Virtual)
	if err != nil {
		s.EventRecorder().Eventf(
			event.Virtual,
			nil,
			"Warning",
			"SyncError",
			fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
			err.Error(),
		)
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.PersistentVolumeClaims.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *persistentVolumeClaimSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.PersistentVolumeClaim]) (result ctrl.Result, retErr error) {
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
		preserveDeletingHostPVC, err := s.shouldPreserveDataProtectionNoDataRestorePVCWhileHostDeleting(ctx, event.Virtual)
		if err != nil {
			return ctrl.Result{}, err
		}
		if preserveDeletingHostPVC {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
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
	recreateHostPVC, err := s.shouldRecreateDataProtectionHostNoDataRestorePVC(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}
	if recreateHostPVC {
		return deleteDataProtectionNoDataRestoreHostPVC(ctx, event.Host, event.Virtual)
	}

	// make sure the persistent volume is synced / faked
	if event.Host.Spec.VolumeName != "" {
		requeue, err := s.ensurePersistentVolume(ctx, event.Host, event.Virtual, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		requeue, err := s.ensureDataProtectionPopulatedPersistentVolumeName(ctx, event.Host, event.Virtual, ctx.Log)
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

		if kerrors.IsConflict(retErr) {
			result = ctrl.Result{RequeueAfter: time.Second}
			retErr = nil
			return
		}

		if retErr != nil {
			s.EventRecorder().Eventf(
				event.Virtual,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
				"Error syncing: %v",
				retErr,
			)
		}
	}()

	// check backwards update
	s.translateUpdateBackwards(event.Host, event.Virtual)

	// copy host status
	vPV, preserveVirtualStatus, err := s.dataProtectionPopulatedPersistentVolume(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}
	if preserveVirtualStatus {
		err = s.ensureDataProtectionHostMaterialization(ctx, event.Host, event.Virtual, vPV)
		if err != nil {
			return ctrl.Result{}, err
		}
		ensureDataProtectionVirtualPopulateStatus(event.Virtual, vPV)
	} else {
		preserveExternalPopulatorStatus, err := s.shouldPreserveExternalPopulatorVirtualStatus(ctx, event.Host, event.Virtual)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !preserveExternalPopulatorStatus {
			event.Virtual.Status = *event.Host.Status.DeepCopy()
		}
	}

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

func (s *persistentVolumeClaimSyncer) translateDataProtectionBackupToHost(ctx *synccontext.SyncContext, vObj *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, bool, error) {
	noDataRestore, err := s.isDataProtectionNoDataRestorePVC(ctx, vObj)
	if err != nil {
		return nil, true, err
	}
	if !noDataRestore {
		return nil, false, nil
	}

	pObj, err := s.translate(ctx, vObj)
	if err != nil {
		return nil, true, err
	}

	clearDataProtectionHostDataSource(pObj)
	pObj.Spec.VolumeName = ""
	return pObj, true, nil
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
				s.EventRecorder().Eventf(
					vObj,
					nil,
					corev1.EventTypeWarning,
					"VolumeNameDiffers",
					fmt.Sprintf("Sync%s", vObj.GetObjectKind().GroupVersionKind().Kind),
					"recreate persistent volume claim because volumeName differs between physical and virtual pvc: %s != %s",
					vObj.Spec.VolumeName,
					newVolumeName,
				)
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

			// The direct update changes the virtual PVC resourceVersion. Stop this
			// reconcile here so the following status patch uses a fresh object.
			return true, nil
		}
	}

	return false, nil
}

func (s *persistentVolumeClaimSyncer) ensureDataProtectionPopulatedPersistentVolumeName(ctx *synccontext.SyncContext, pObj *corev1.PersistentVolumeClaim, vObj *corev1.PersistentVolumeClaim, log loghelper.Logger) (bool, error) {
	if vObj.Spec.VolumeName != "" || !isDataProtectionBackupPVC(vObj) || !isHostPVCWaitingForVolume(pObj) {
		return false, nil
	}

	vPV, ok, err := s.findDataProtectionPopulatedPersistentVolumeByClaimRef(ctx, vObj)
	if err != nil || !ok {
		return false, err
	}

	log.Infof("update virtual data protection pvc %s/%s volume name to populated pv %s", vObj.Namespace, vObj.Name, vPV.Name)
	vObj.Spec.VolumeName = vPV.Name
	err = ctx.VirtualClient.Update(ctx, vObj)
	if err != nil {
		return false, err
	}

	// The direct update changes the virtual PVC resourceVersion. Stop this
	// reconcile here so the following status patch uses a fresh object.
	return true, nil
}

func (s *persistentVolumeClaimSyncer) findDataProtectionPopulatedPersistentVolumeByClaimRef(ctx *synccontext.SyncContext, vObj *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, bool, error) {
	vPVs := &corev1.PersistentVolumeList{}
	err := ctx.VirtualClient.List(ctx.Context, vPVs)
	if err != nil {
		return nil, false, err
	}

	var match *corev1.PersistentVolume
	for i := range vPVs.Items {
		vPV := &vPVs.Items[i]
		if !isDataProtectionPopulatedPersistentVolumeForPVC(vPV, vObj, true) {
			continue
		}
		if match != nil && match.Name != vPV.Name {
			return nil, false, fmt.Errorf("multiple data protection populated persistent volumes match pvc %s/%s", vObj.Namespace, vObj.Name)
		}
		match = vPV.DeepCopy()
	}

	if match == nil {
		return nil, false, nil
	}

	return match, true, nil
}

func (s *persistentVolumeClaimSyncer) ensureDataProtectionHostMaterialization(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim, vPV *corev1.PersistentVolume) error {
	hostPVName := s.dataProtectionHostPersistentVolumeName(ctx, vPV)
	hostPV := &corev1.PersistentVolume{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{Name: hostPVName}, hostPV)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return s.upsertDataProtectionMaterializationRequest(ctx, pObj, vObj, vPV)
		}
		return err
	}

	helperPVC, helperFound, err := s.findDataProtectionPopulateHelperPVC(ctx, vObj, vPV)
	if err != nil {
		return err
	}

	pObj.Spec.VolumeName = hostPVName
	return s.ensureDataProtectionHostPVClaimRef(ctx, hostPVName, pObj, helperPVC, helperFound)
}

func (s *persistentVolumeClaimSyncer) upsertDataProtectionMaterializationRequest(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim, vPV *corev1.PersistentVolume) error {
	desired := dataProtectionMaterializationRequest(ctx.Config.HostNamespace, pObj, vObj, vPV)
	existing := &corev1.ConfigMap{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{
		Namespace: desired.Namespace,
		Name:      desired.Name,
	}, existing)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctx.HostClient.Create(ctx.Context, desired)
		}
		return err
	}

	updated := existing.DeepCopy()
	updated.Labels = desired.Labels
	updated.Data = desired.Data
	if equality.Semantic.DeepEqual(existing.Labels, updated.Labels) &&
		equality.Semantic.DeepEqual(existing.Data, updated.Data) {
		return nil
	}

	return ctx.HostClient.Patch(ctx.Context, updated, client.MergeFrom(existing))
}

func (s *persistentVolumeClaimSyncer) dataProtectionHostPersistentVolumeName(ctx *synccontext.SyncContext, vPV *corev1.PersistentVolume) string {
	if s.useFakePersistentVolumes {
		return vPV.Name
	}

	return mappings.VirtualToHostName(ctx, vPV.Name, "", mappings.PersistentVolumes())
}

func (s *persistentVolumeClaimSyncer) findDataProtectionPopulateHelperPVC(ctx *synccontext.SyncContext, vObj *corev1.PersistentVolumeClaim, vPV *corev1.PersistentVolume) (*corev1.PersistentVolumeClaim, bool, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := ctx.VirtualClient.List(ctx.Context, pvcList, client.InNamespace(vObj.Namespace))
	if err != nil {
		return nil, false, err
	}

	var match *corev1.PersistentVolumeClaim
	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		if pvc.Name == vObj.Name || pvc.Spec.VolumeName != vPV.Name {
			continue
		}
		if match != nil && match.Name != pvc.Name {
			return nil, false, fmt.Errorf("multiple data protection helper persistent volume claims match pv %s for pvc %s/%s", vPV.Name, vObj.Namespace, vObj.Name)
		}
		match = pvc.DeepCopy()
	}
	if match == nil {
		return nil, false, nil
	}

	return match, true, nil
}

func (s *persistentVolumeClaimSyncer) ensureDataProtectionHostPVClaimRef(ctx *synccontext.SyncContext, hostPVName string, pObj *corev1.PersistentVolumeClaim, helperPVC *corev1.PersistentVolumeClaim, helperFound bool) error {
	hostPV := &corev1.PersistentVolume{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{Name: hostPVName}, hostPV)
	if err != nil {
		return err
	}

	targetRef := &corev1.ObjectReference{
		APIVersion: corev1.SchemeGroupVersion.Version,
		Kind:       "PersistentVolumeClaim",
		Namespace:  pObj.Namespace,
		Name:       pObj.Name,
		UID:        pObj.UID,
	}
	if claimRefReferencesPersistentVolumeClaim(hostPV.Spec.ClaimRef, pObj) {
		if hostPV.Spec.ClaimRef.UID == pObj.UID {
			return nil
		}
	} else if hostPV.Spec.ClaimRef != nil {
		if !helperFound {
			return fmt.Errorf("host pv %s is bound to %s/%s, but no virtual populate helper pvc was found for target pvc %s/%s", hostPVName, hostPV.Spec.ClaimRef.Namespace, hostPV.Spec.ClaimRef.Name, pObj.Namespace, pObj.Name)
		}
		ok, err := s.hostPVClaimRefMatchesVirtualPVC(ctx, hostPV.Spec.ClaimRef, helperPVC)
		if err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("host pv %s claimRef %s/%s does not match target pvc %s/%s or expected populate helper pvc %s/%s", hostPVName, hostPV.Spec.ClaimRef.Namespace, hostPV.Spec.ClaimRef.Name, pObj.Namespace, pObj.Name, helperPVC.Namespace, helperPVC.Name)
		}
	}

	updated := hostPV.DeepCopy()
	updated.Spec.ClaimRef = targetRef
	return ctx.HostClient.Patch(ctx.Context, updated, client.MergeFrom(hostPV))
}

func (s *persistentVolumeClaimSyncer) hostPVClaimRefMatchesVirtualPVC(ctx *synccontext.SyncContext, ref *corev1.ObjectReference, vPVC *corev1.PersistentVolumeClaim) (bool, error) {
	if ref == nil || vPVC == nil {
		return false, nil
	}

	hostPVCName := s.VirtualToHost(ctx, types.NamespacedName{Name: vPVC.Name, Namespace: vPVC.Namespace}, vPVC)
	if ref.Namespace != hostPVCName.Namespace || ref.Name != hostPVCName.Name {
		return false, nil
	}

	hostPVC := &corev1.PersistentVolumeClaim{}
	err := ctx.HostClient.Get(ctx.Context, hostPVCName, hostPVC)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return ref.UID == "" || ref.UID == hostPVC.UID, nil
}

func claimRefMatchesPersistentVolumeClaim(ref *corev1.ObjectReference, pvc *corev1.PersistentVolumeClaim) bool {
	if ref == nil || pvc == nil {
		return false
	}

	return claimRefReferencesPersistentVolumeClaim(ref, pvc) &&
		(ref.UID == "" || ref.UID == pvc.UID)
}

func claimRefReferencesPersistentVolumeClaim(ref *corev1.ObjectReference, pvc *corev1.PersistentVolumeClaim) bool {
	if ref == nil || pvc == nil {
		return false
	}

	return ref.Namespace == pvc.Namespace && ref.Name == pvc.Name
}

func (s *persistentVolumeClaimSyncer) dataProtectionPopulatedPersistentVolume(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, bool, error) {
	if !isDataProtectionBackupPVC(vObj) || vObj.Spec.VolumeName == "" || !isHostPVCWaitingForVolume(pObj) {
		return nil, false, nil
	}

	vPV := &corev1.PersistentVolume{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: vObj.Spec.VolumeName}, vPV)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	if vPV.Annotations[dataProtectionPopulateFromAnnotation] == "" {
		return nil, false, nil
	}
	if !isDataProtectionPopulatedPersistentVolumeForPVC(vPV, vObj, false) {
		return nil, false, nil
	}
	if !isVirtualPVCBound(vObj) && vPV.Status.Phase != corev1.VolumeBound {
		return nil, false, nil
	}

	return vPV, true, nil
}

func (s *persistentVolumeClaimSyncer) shouldPreserveExternalPopulatorVirtualStatus(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim) (bool, error) {
	if !hasExternalPopulatorDataSource(vObj) {
		return false, nil
	}
	if !isDataProtectionBackupPVC(vObj) {
		return true, nil
	}
	if !isHostPVCWaitingForVolume(pObj) {
		return false, nil
	}
	if vObj.Spec.VolumeName == "" {
		return true, nil
	}

	vPV := &corev1.PersistentVolume{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: vObj.Spec.VolumeName}, vPV)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	if vPV.Annotations[dataProtectionPopulateFromAnnotation] == "" {
		return true, nil
	}

	return isDataProtectionPopulatedPersistentVolumeForPVC(vPV, vObj, false), nil
}

func ensureDataProtectionVirtualPopulateStatus(vObj *corev1.PersistentVolumeClaim, vPV *corev1.PersistentVolume) {
	if isVirtualPVCBound(vObj) {
		return
	}

	if vPV.Status.Phase != corev1.VolumeBound {
		return
	}

	vObj.Status.Phase = corev1.ClaimBound
	if len(vObj.Status.AccessModes) == 0 {
		vObj.Status.AccessModes = append([]corev1.PersistentVolumeAccessMode(nil), vPV.Spec.AccessModes...)
	}
	storage, ok := vObj.Status.Capacity[corev1.ResourceStorage]
	if vObj.Status.Capacity == nil || !ok || storage.IsZero() {
		vObj.Status.Capacity = vPV.Spec.Capacity.DeepCopy()
	}
}

func isDataProtectionPopulatedPersistentVolumeForPVC(vPV *corev1.PersistentVolume, vObj *corev1.PersistentVolumeClaim, requireBoundPV bool) bool {
	if requireBoundPV && vPV.Status.Phase != corev1.VolumeBound {
		return false
	}
	if vPV.Annotations[dataProtectionPopulateFromAnnotation] == "" {
		return false
	}
	if vPV.Spec.ClaimRef == nil ||
		vPV.Spec.ClaimRef.Namespace != vObj.Namespace ||
		vPV.Spec.ClaimRef.Name != vObj.Name {
		return false
	}
	if vPV.Spec.ClaimRef.UID != "" && vPV.Spec.ClaimRef.UID != vObj.UID {
		return false
	}

	return true
}

func dataProtectionMaterializationRequest(hostNamespace string, pObj, vObj *corev1.PersistentVolumeClaim, vPV *corev1.PersistentVolume) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hostNamespace,
			Name:      dataProtectionMaterializationRequestName(pObj),
			Labels: map[string]string{
				dataProtectionMaterializationRequestLabel: "true",
			},
		},
		Data: map[string]string{
			"state":               dataProtectionMaterializationStatePending,
			"hostPVCNamespace":    pObj.Namespace,
			"hostPVCName":         pObj.Name,
			"virtualPVCNamespace": vObj.Namespace,
			"virtualPVCName":      vObj.Name,
			"virtualPVCUID":       string(vObj.UID),
			"virtualPVName":       vPV.Name,
			"virtualPVUID":        string(vPV.UID),
			"backupName":          vObj.Spec.DataSourceRef.Name,
			"populateFrom":        vPV.Annotations[dataProtectionPopulateFromAnnotation],
		},
	}
}

func dataProtectionMaterializationRequestName(pObj *corev1.PersistentVolumeClaim) string {
	sum := sha256.Sum256([]byte(pObj.Namespace + "/" + pObj.Name))
	return dataProtectionMaterializationRequestPrefix + hex.EncodeToString(sum[:])[:16]
}

func isDataProtectionBackupPVC(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Spec.DataSourceRef == nil || pvc.Spec.DataSourceRef.APIGroup == nil {
		return false
	}

	return *pvc.Spec.DataSourceRef.APIGroup == dataProtectionAPIGroup &&
		pvc.Spec.DataSourceRef.Kind == dataProtectionBackupKind &&
		pvc.Spec.DataSourceRef.Name != ""
}

func hasExternalPopulatorDataSource(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Spec.DataSourceRef == nil {
		return false
	}

	switch pvc.Spec.DataSourceRef.Kind {
	case "VolumeSnapshot", "PersistentVolumeClaim":
		return false
	default:
		return true
	}
}

func isVirtualPVCBound(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Spec.VolumeName == "" || pvc.Status.Phase != corev1.ClaimBound {
		return false
	}

	storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]
	return ok && !storage.IsZero()
}

func isHostPVCWaitingForVolume(pvc *corev1.PersistentVolumeClaim) bool {
	if pvc.Status.Phase == corev1.ClaimBound {
		return false
	}

	storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]
	return !ok || storage.IsZero()
}

func isDataProtectionRestoreProvisionedWithoutDataRestore(pvc *corev1.PersistentVolumeClaim) bool {
	for _, condition := range pvc.Status.Conditions {
		if condition.Type == corev1.PersistentVolumeClaimConditionType("Restore") &&
			condition.Status == corev1.ConditionTrue &&
			condition.Reason == dataProtectionRestoreConditionReasonProvisioned {
			return true
		}
	}

	return false
}

func clearDataProtectionHostDataSource(pvc *corev1.PersistentVolumeClaim) {
	pvc.Spec.DataSource = nil
	pvc.Spec.DataSourceRef = nil
}

func (s *persistentVolumeClaimSyncer) shouldRecreateDataProtectionHostNoDataRestorePVC(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim) (bool, error) {
	if !isHostPVCWaitingForVolume(pObj) ||
		(!isDataProtectionBackupDataSource(pObj.Spec.DataSource) && !isDataProtectionBackupDataSourceRef(pObj.Spec.DataSourceRef)) {
		return false, nil
	}

	noDataRestore, err := s.isDataProtectionNoDataRestorePVC(ctx, vObj)
	if err != nil || !noDataRestore {
		return false, err
	}

	return true, nil
}

func (s *persistentVolumeClaimSyncer) shouldPreserveDataProtectionNoDataRestorePVCWhileHostDeleting(ctx *synccontext.SyncContext, vObj *corev1.PersistentVolumeClaim) (bool, error) {
	return s.isDataProtectionNoDataRestorePVC(ctx, vObj)
}

func (s *persistentVolumeClaimSyncer) isDataProtectionNoDataRestorePVC(ctx *synccontext.SyncContext, vObj *corev1.PersistentVolumeClaim) (bool, error) {
	if !isDataProtectionBackupPVC(vObj) {
		return false, nil
	}

	return isDataProtectionRestoreProvisionedWithoutDataRestore(vObj), nil
}

func deleteDataProtectionNoDataRestoreHostPVC(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim) (ctrl.Result, error) {
	result, err := patcher.DeleteHostObjectWithOptions(ctx, pObj, vObj, "data protection restore pvc was provisioned without data restore", &client.DeleteOptions{
		GracePeriodSeconds: &zero,
		Preconditions:      hostDeletePreconditions(pObj),
	})
	if kerrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}

	return result, err
}

func hostDeletePreconditions(obj client.Object) *metav1.Preconditions {
	preconditions := &metav1.Preconditions{}
	if obj.GetUID() != "" {
		uid := obj.GetUID()
		preconditions.UID = &uid
	}
	if obj.GetResourceVersion() != "" {
		resourceVersion := obj.GetResourceVersion()
		preconditions.ResourceVersion = &resourceVersion
	}
	if preconditions.UID == nil && preconditions.ResourceVersion == nil {
		return nil
	}

	return preconditions
}

func isDataProtectionBackupDataSource(ref *corev1.TypedLocalObjectReference) bool {
	if ref == nil || ref.APIGroup == nil {
		return false
	}

	return *ref.APIGroup == dataProtectionAPIGroup &&
		ref.Kind == dataProtectionBackupKind &&
		ref.Name != ""
}

func isDataProtectionBackupDataSourceRef(ref *corev1.TypedObjectReference) bool {
	if ref == nil || ref.APIGroup == nil {
		return false
	}

	return *ref.APIGroup == dataProtectionAPIGroup &&
		ref.Kind == dataProtectionBackupKind &&
		ref.Name != ""
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
		s.EventRecorder().Eventf(
			virtual,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", virtual.GetObjectKind().GroupVersionKind().Kind),
			"did not sync persistent volume claim %q to host because the storage class %q couldn't be reached in the host: %s",
			virtual.GetName(),
			*virtual.Spec.StorageClassName,
			err,
		)
		return true
	}
	matches, err := ctx.Config.Sync.FromHost.StorageClasses.Selector.Matches(pStorageClass)
	if err != nil {
		s.EventRecorder().Eventf(
			virtual,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", virtual.GetObjectKind().GroupVersionKind().Kind),
			"did not sync persistent volume claim %q to host because the storage class %q in the host could not be checked against the selector under 'sync.fromHost.storageClasses.selector': %s",
			virtual.GetName(),
			pStorageClass.GetName(),
			err,
		)
		return true
	}
	if !matches {
		s.EventRecorder().Eventf(
			virtual,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", virtual.GetObjectKind().GroupVersionKind().Kind),
			"did not sync persistent volume claim %q to host because the storage class %q in the host does not match the selector under 'sync.fromHost.storageClasses.selector'",
			virtual.GetName(),
			pStorageClass.GetName(),
		)
		return true
	}

	return false
}
