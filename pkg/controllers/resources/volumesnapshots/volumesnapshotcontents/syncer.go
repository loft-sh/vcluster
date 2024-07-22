package volumesnapshotcontents

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	translator2 "github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PhysicalVSCGarbageCollectionFinalizer = "vcluster.loft.sh/physical-volumesnapshotcontent-gc"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.VolumeSnapshotContents())
	if err != nil {
		return nil, err
	}

	return &volumeSnapshotContentSyncer{
		GenericTranslator: translator2.NewGenericTranslator(ctx, "volume-snapshot-content", &volumesnapshotv1.VolumeSnapshotContent{}, mapper),

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type volumeSnapshotContentSyncer struct {
	syncertypes.GenericTranslator

	virtualClient client.Client
}

var _ syncertypes.ToVirtualSyncer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore this VolumeSnapshotContent resource, because there is no virtual VolumeSnapshot bound to it
		return ctrl.Result{}, nil
	}

	vVSC := s.translateBackwards(pVSC, vVS)
	ctx.Log.Infof("create VolumeSnapshotContent %s, because it does not exist in the virtual cluster", vVSC.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vVSC)
}

var _ syncertypes.Syncer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if ctx.IsDelete || vVSC.DeletionTimestamp != nil || (vVSC.Annotations != nil && vVSC.Annotations[constants.HostClusterVSCAnnotation] != "") {
		if len(vVSC.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vVSC.Finalizers = []string{}
			ctx.Log.Infof("remove virtual VolumeSnapshotContent %s finalizers, because object should get deleted", vVSC.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx, vVSC)
		}

		ctx.Log.Infof("remove virtual VolumeSnapshotContent %s, because object should get deleted", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vVSC)
	}

	pVSC := s.translate(ctx, vVSC)
	ctx.Log.Infof("create physical VolumeSnapshotContent %s, because there is a virtual VolumeSnapshotContent", pVSC.Name)
	err := ctx.PhysicalClient.Create(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotContentSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)

	// check if objects are getting deleted
	if vObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pObj.GetName())
			err := ctx.PhysicalClient.Delete(ctx, pObj)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// sync finalizers and status to allow tracking of the deletion progress

		// TODO: refactor finalizer syncing and handling
		// we can not add new finalizers from physical to virtual once it has deletionTimestamp, we can only remove finalizers

		if !equality.Semantic.DeepEqual(vVSC.Finalizers, pVSC.Finalizers) {
			updated := vVSC.DeepCopy()
			updated.Finalizers = pVSC.Finalizers
			ctx.Log.Infof("update finalizers of the virtual VolumeSnapshotContent %s, because finalizers on the physical resource changed", vVSC.Name)
			err := s.virtualClient.Update(ctx, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		if !equality.Semantic.DeepEqual(vVSC.Status, pVSC.Status) {
			updated := vVSC.DeepCopy()
			updated.Status = pVSC.Status.DeepCopy()
			ctx.Log.Infof("update virtual VolumeSnapshotContent %s, because status has changed", vVSC.Name)
			err := s.virtualClient.Status().Update(ctx, updated)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the VolumeSnapshotContent should get synced
	sync, _, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore VolumeSnapshotContent object for which VolumeSnapshot was deleted,
		// it will be automatically managed by the snapshot controller based on deletion policy
		return ctrl.Result{}, nil
	}

	// update the physical VolumeSnapshotContent if the virtual has changed
	if vVSC.Annotations[constants.HostClusterVSCAnnotation] == "" && vVSC.DeletionTimestamp != nil {
		if pVSC.DeletionTimestamp != nil {
			return ctrl.Result{}, nil
		}

		ctx.Log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pVSC.Name)
		err := ctx.PhysicalClient.Delete(ctx, pVSC, &client.DeleteOptions{
			GracePeriodSeconds: vVSC.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pVSC.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, pVSC, vVSC)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pVSC, vVSC); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// update virtual object
	s.translateUpdateBackwards(pVSC, vVSC)

	// update virtual status
	vVSC.Status = pVSC.Status.DeepCopy()

	// update host object
	if vVSC.Annotations[constants.HostClusterVSCAnnotation] == "" {
		pVSC.Spec.DeletionPolicy = vVSC.Spec.DeletionPolicy
		pVSC.Spec.VolumeSnapshotClassName = vVSC.Spec.VolumeSnapshotClassName
		pVSC.Annotations, pVSC.Labels = translate.HostAnnotations(vVSC, pVSC), translate.HostLabels(ctx, vVSC, pVSC)
	}

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotContentSyncer) shouldSync(ctx *synccontext.SyncContext, pObj *volumesnapshotv1.VolumeSnapshotContent) (bool, *volumesnapshotv1.VolumeSnapshot, error) {
	vName := mappings.HostToVirtual(ctx, pObj.Spec.VolumeSnapshotRef.Name, pObj.Spec.VolumeSnapshotRef.Namespace, nil, mappings.VolumeSnapshots())
	if vName.Name == "" {
		return false, nil, nil
	}

	vVS := &volumesnapshotv1.VolumeSnapshot{}
	err := s.virtualClient.Get(ctx, vName, vVS)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.Default.IsManaged(ctx, pObj) {
			return true, vVS, nil
		}
		return false, nil, nil
	}

	return true, vVS, nil
}

func (s *volumeSnapshotContentSyncer) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	pVSC, ok := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok {
		return false, nil
	}

	sync, _, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return false, nil
	}

	return sync, nil
}
