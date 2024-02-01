package volumesnapshotcontents

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HostClusterVSCAnnotation              = "vcluster.loft.sh/host-volumesnapshotcontent"
	PhysicalVSCGarbageCollectionFinalizer = "vcluster.loft.sh/physical-volumesnapshotcontent-gc"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &volumeSnapshotContentSyncer{
		Translator: translator.NewClusterTranslator(ctx, "volume-snapshot-content", &volumesnapshotv1.VolumeSnapshotContent{}, NewVolumeSnapshotContentTranslator()),

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type volumeSnapshotContentSyncer struct {
	translator.Translator

	virtualClient client.Client
}

var _ syncer.Initializer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) Init(registerContext *synccontext.RegisterContext) error {
	return util.EnsureCRD(registerContext.Context, registerContext.VirtualManager.GetConfig(), []byte(volumeSnapshotContentsCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"))
}

func NewVolumeSnapshotContentTranslator() translate.PhysicalNameTranslator {
	return func(vName string, vObj client.Object) string {
		return translateVolumeSnapshotContentName(vName, vObj)
	}
}

var _ syncer.IndicesRegisterer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &volumesnapshotv1.VolumeSnapshotContent{}, constants.IndexByPhysicalName, newIndexByVSCPhysicalName())
}

func newIndexByVSCPhysicalName() client.IndexerFunc {
	return func(rawObj client.Object) []string {
		return []string{translateVolumeSnapshotContentName(rawObj.GetName(), rawObj)}
	}
}

var _ syncer.ToVirtualSyncer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx.Context, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore this VolumeSnapshotContent resource, because there is no virtual VolumeSnapshot bound to it
		return ctrl.Result{}, nil
	}

	vVSC := s.translateBackwards(pVSC, vVS)
	ctx.Log.Infof("create VolumeSnapshotContent %s, because it does not exist in the virtual cluster", vVSC.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx.Context, vVSC)
}

var _ syncer.Syncer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if vVSC.DeletionTimestamp != nil || (vVSC.Annotations != nil && vVSC.Annotations[HostClusterVSCAnnotation] != "") {
		if len(vVSC.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vVSC.Finalizers = []string{}
			ctx.Log.Infof("remove virtual VolumeSnapshotContent %s finalizers, because object should get deleted", vVSC.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx.Context, vVSC)
		}

		ctx.Log.Infof("remove virtual VolumeSnapshotContent %s, because object should get deleted", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx.Context, vVSC)
	}

	pVSC := s.translate(ctx.Context, vVSC)
	ctx.Log.Infof("create physical VolumeSnapshotContent %s, because there is a virtual VolumeSnapshotContent", pVSC.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotContentSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)

	// check if objects are getting deleted
	if vObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pObj.GetName())
			err := ctx.PhysicalClient.Delete(ctx.Context, pObj)
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
			translator.PrintChanges(vObj, updated, ctx.Log)
			err := s.virtualClient.Update(ctx.Context, updated)
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
			translator.PrintChanges(vObj, updated, ctx.Log)
			err := s.virtualClient.Status().Update(ctx.Context, updated)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx.Context, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore VolumeSnapshotContent object for which VolumeSnapshot was deleted,
		// it will be automatically managed by the snapshot controller based on deletion policy
		return ctrl.Result{}, nil
	}

	updatedObj := s.translateUpdateBackwards(pVSC, vVSC, vVS)
	if updatedObj != nil {
		ctx.Log.Infof("update virtual VolumeSnapshotContent %s, because spec or metadata(annotations or labels) have changed", vVSC.Name)
		translator.PrintChanges(vObj, updatedObj, ctx.Log)
		return ctrl.Result{}, s.virtualClient.Update(ctx.Context, updatedObj)
	}

	// update virtual status if it differs
	if !equality.Semantic.DeepEqual(vVSC.Status, pVSC.Status) {
		newVSC := vVSC.DeepCopy()
		newVSC.Status = pVSC.Status.DeepCopy()
		translator.PrintChanges(vObj, newVSC, ctx.Log)
		ctx.Log.Infof("update virtual VolumeSnapshotContent %s, because status has changed", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Status().Update(ctx.Context, newVSC)
	}

	// update the physical VolumeSnapshotContent if the virtual has changed
	if vVSC.Annotations == nil || vVSC.Annotations[HostClusterVSCAnnotation] == "" {
		if vVSC.DeletionTimestamp != nil {
			if pVSC.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}

			ctx.Log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pVSC.Name)
			err := ctx.PhysicalClient.Delete(ctx.Context, pVSC, &client.DeleteOptions{
				GracePeriodSeconds: vVSC.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pVSC.UID)),
			})
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		updatedPv := s.translateUpdate(ctx.Context, vVSC, pVSC)
		if updatedPv != nil {
			ctx.Log.Infof("update physical VolumeSnapshotContent %s, because spec or annotations have changed", updatedPv.Name)
			translator.PrintChanges(pVSC, updatedPv, ctx.Log)
			err := ctx.PhysicalClient.Update(ctx.Context, updatedPv)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotContentSyncer) shouldSync(ctx context.Context, pObj *volumesnapshotv1.VolumeSnapshotContent) (bool, *volumesnapshotv1.VolumeSnapshot, error) {
	vVS := &volumesnapshotv1.VolumeSnapshot{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vVS, constants.IndexByPhysicalName, pObj.Spec.VolumeSnapshotRef.Namespace+"/"+pObj.Spec.VolumeSnapshotRef.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.Default.IsManagedCluster(pObj) {
			return true, nil, nil
		}
		return false, nil, nil
	}

	return true, vVS, nil
}

func (s *volumeSnapshotContentSyncer) IsManaged(ctx context.Context, pObj client.Object) (bool, error) {
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

func (s *volumeSnapshotContentSyncer) VirtualToHost(_ context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translateVolumeSnapshotContentName(req.Name, vObj)}
}

func (s *volumeSnapshotContentSyncer) HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Name: pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := &volumesnapshotv1.VolumeSnapshotContent{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: req.Name}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translateVolumeSnapshotContentName(name string, vObj runtime.Object) string {
	if vObj == nil {
		return name
	}

	vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok || vVSC.Annotations == nil || vVSC.Annotations[HostClusterVSCAnnotation] == "" {
		return translate.Default.PhysicalNameClusterScoped(name)
	}

	return vVSC.Annotations[HostClusterVSCAnnotation]
}
