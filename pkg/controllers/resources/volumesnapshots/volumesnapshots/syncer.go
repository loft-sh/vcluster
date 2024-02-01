package volumesnapshots

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &volumeSnapshotSyncer{
		NamespacedTranslator:                translator.NewNamespacedTranslator(ctx, "volume-snapshot", &volumesnapshotv1.VolumeSnapshot{}),
		volumeSnapshotContentNameTranslator: volumesnapshotcontents.NewVolumeSnapshotContentTranslator(),
	}, nil
}

type volumeSnapshotSyncer struct {
	translator.NamespacedTranslator
	volumeSnapshotContentNameTranslator translate.PhysicalNameTranslator
}

var _ syncer.Initializer = &volumeSnapshotSyncer{}

func (s *volumeSnapshotSyncer) Init(registerContext *synccontext.RegisterContext) error {
	return util.EnsureCRD(registerContext.Context, registerContext.VirtualManager.GetConfig(), []byte(volumeSnapshotCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"))
}

var _ syncer.Syncer = &volumeSnapshotSyncer{}

func (s *volumeSnapshotSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vVS := vObj.(*volumesnapshotv1.VolumeSnapshot)
	if vVS.DeletionTimestamp != nil {
		// delete volume snapshot immediately
		if len(vObj.GetFinalizers()) > 0 || (vObj.GetDeletionGracePeriodSeconds() != nil && *vObj.GetDeletionGracePeriodSeconds() > 0) {
			vObj.SetFinalizers([]string{})
			vObj.SetDeletionGracePeriodSeconds(&zero)
			return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, vObj)
		}
		return ctrl.Result{}, nil
	}

	pObj, err := s.translate(ctx, vVS)
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.SyncToHostCreate(ctx, vObj, pObj)
}

func (s *volumeSnapshotSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vVS := vObj.(*volumesnapshotv1.VolumeSnapshot)
	pVS := pObj.(*volumesnapshotv1.VolumeSnapshot)

	if pVS.DeletionTimestamp != nil {
		if vVS.DeletionTimestamp == nil {
			ctx.Log.Infof("delete virtual volume snapshot %s/%s, because the physical volume snapshot is being deleted", vVS.Namespace, vVS.Name)
			err := ctx.VirtualClient.Delete(ctx.Context, vVS, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if *vVS.DeletionGracePeriodSeconds != *pVS.DeletionGracePeriodSeconds {
			ctx.Log.Infof("delete virtual volume snapshot %s/%s with grace period seconds %v", vVS.Namespace, vVS.Name, *pVS.DeletionGracePeriodSeconds)
			err := ctx.VirtualClient.Delete(ctx.Context, vVS, &client.DeleteOptions{GracePeriodSeconds: pVS.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vVS.UID))})
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// sync finalizers and status to allow tracking of the deletion progress
		//TODO: refactor finalizer syncing and handling
		// we can not add new finalizers from physical to virtual once it has deletionTimestamp, we can only remove finalizers

		if !equality.Semantic.DeepEqual(vVS.Finalizers, pVS.Finalizers) {
			updated := vVS.DeepCopy()
			updated.Finalizers = pVS.Finalizers
			ctx.Log.Infof("update finalizers of the virtual VolumeSnapshot %s, because finalizers on the physical resource changed", vVS.Name)
			translator.PrintChanges(vObj, updated, ctx.Log)
			err := ctx.VirtualClient.Update(ctx.Context, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		if !equality.Semantic.DeepEqual(vVS.Status, pVS.Status) {
			updated := vVS.DeepCopy()
			updated.Status = pVS.Status.DeepCopy()
			ctx.Log.Infof("update virtual VolumeSnapshot %s, because status has changed", vVS.Name)
			translator.PrintChanges(vObj, updated, ctx.Log)
			err := ctx.VirtualClient.Status().Update(ctx.Context, updated)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	} else if vVS.DeletionTimestamp != nil {
		if pVS.DeletionTimestamp == nil {
			ctx.Log.Infof("delete physical volume snapshot %s/%s, because virtual volume snapshot is being deleted", pVS.Namespace, pVS.Name)
			return ctrl.Result{}, ctx.PhysicalClient.Delete(ctx.Context, pVS, &client.DeleteOptions{
				GracePeriodSeconds: vVS.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pVS.UID)),
			})
		}
		return ctrl.Result{}, nil
	}

	// check backwards update
	updated := s.translateUpdateBackwards(pVS, vVS)
	if updated != nil {
		ctx.Log.Infof("update virtual volume snapshot %s/%s, because the spec has changed", vVS.Namespace, vVS.Name)
		translator.PrintChanges(vObj, updated, ctx.Log)
		err := ctx.VirtualClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// check backwards status
	if !equality.Semantic.DeepEqual(vVS.Status, pVS.Status) {
		updated := vVS.DeepCopy()
		updated.Status = pVS.Status.DeepCopy()
		ctx.Log.Infof("update virtual volume snapshot %s/%s, because the status has changed", vVS.Namespace, vVS.Name)
		translator.PrintChanges(vObj, updated, ctx.Log)
		err := ctx.VirtualClient.Status().Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// forward update
	updated = s.translateUpdate(ctx.Context, pVS, vVS)
	if updated != nil {
		translator.PrintChanges(pVS, updated, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vVS, updated)
}
