package volumesnapshotcontents

import (
	"context"
	"time"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HostClusterVSCAnnotation              = "vcluster.loft.sh/host-volumesnapshotcontent"
	PhysicalVSCGarbageCollectionFinalizer = "vcluster.loft.sh/physical-volumesnapshotcontent-gc"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &volumesnapshotv1.VolumeSnapshotContent{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translateVolumeSnapshotContentName(ctx.Options.TargetNamespace, rawObj.GetName(), rawObj)}
	})
}

func Register(ctx *context2.ControllerContext, _ record.EventBroadcaster) error {
	nameTranslator := NewVolumeSnapshotContentTranslator(ctx.Options.TargetNamespace)
	return generic.RegisterSyncer(ctx, "volumesnapshotcontent", &syncer{
		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),
		translator:      translate.NewDefaultClusterTranslator(ctx.Options.TargetNamespace, nameTranslator),
	})
}

var _ generic.BackwardSyncer = &syncer{}

type syncer struct {
	generic.Translator

	targetNamespace string
	virtualClient   client.Client
	localClient     client.Client

	translator translate.Translator
}

func NewVolumeSnapshotContentTranslator(physicalNamespace string) translate.PhysicalNameTranslator {
	return func(vName string, vObj client.Object) string {
		return translateVolumeSnapshotContentName(physicalNamespace, vName, vObj)
	}
}

func (s *syncer) New() client.Object {
	return &volumesnapshotv1.VolumeSnapshotContent{}
}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore this VolumeSnapshotContent resource, because there is no virtual VolumeSnapshot bound to it"
		return ctrl.Result{}, nil
	}

	vVSC := s.translateBackwards(pVSC, vVS)
	log.Infof("create VolumeSnapshotContent %s, because it does not exist in the virtual cluster", vVSC.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vVSC)
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if vVSC.DeletionTimestamp != nil || (vVSC.Annotations != nil && vVSC.Annotations[HostClusterVSCAnnotation] != "") {
		if len(vVSC.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vVSC.Finalizers = []string{}
			log.Infof("remove virtual VolumeSnapshotContent %s finalizers, because object should get deleted", vVSC.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx, vVSC)
		}

		log.Infof("remove virtual VolumeSnapshotContent %s, because object should get deleted", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vVSC)
	}

	pVSC, err := s.translate(vVSC)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical VolumeSnapshotContent %s, because there is a virtual VolumeSnapshotContent", pVSC.Name)
	err = s.localClient.Create(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pVSC := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	vVSC := vObj.(*volumesnapshotv1.VolumeSnapshotContent)

	//check if objects are getting deleted
	if vObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pObj.GetName())
			err := s.localClient.Delete(ctx, pObj)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// sync finalizers and status to allow tracking of the deletion progress
		if !equality.Semantic.DeepEqual(vVSC.Finalizers, pVSC.Finalizers) {
			updated := vVSC.DeepCopy()
			updated.Finalizers = pVSC.Finalizers
			log.Infof("update finalizers of the virtual VolumeSnapshotContent %s, because finalizers on the physical resource changed", vVSC.Name)
			err := s.virtualClient.Update(ctx, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		if !equality.Semantic.DeepEqual(vVSC.Status, pVSC.Status) {
			vVSC.Status = pVSC.Status.DeepCopy()
			log.Infof("update virtual VolumeSnapshotContent %s, because status has changed", vVSC.Name)
			err := s.virtualClient.Status().Update(ctx, vVSC)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore VolumeSnapshotContent object for which VolumeSnapshot was deleted,
		// it will be automatically managed by the snapshot controller based on deletion policy
		return ctrl.Result{}, nil
	}

	updatedObj := s.translateUpdateBackwards(pVSC, vVSC, vVS)
	if updatedObj != nil {
		log.Infof("update virtual VolumeSnapshotContent %s, because spec or metadata(annotations or labels) have changed", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Update(ctx, updatedObj)
	}

	// update virtual status if it differs
	if !equality.Semantic.DeepEqual(vVSC.Status, pVSC.Status) {
		vVSC.Status = pVSC.Status.DeepCopy()
		log.Infof("update virtual VolumeSnapshotContent %s, because status has changed", vVSC.Name)
		return ctrl.Result{}, s.virtualClient.Status().Update(ctx, vVSC)
	}

	// update the physical VolumeSnapshotContent if the virtual has changed
	if vVSC.Annotations == nil || vVSC.Annotations[HostClusterVSCAnnotation] == "" {
		if vVSC.DeletionTimestamp != nil {
			if pVSC.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}

			log.Infof("delete physical VolumeSnapshotContent %s, because virtual VolumeSnapshotContent is being deleted", pVSC.Name)
			err := s.localClient.Delete(ctx, pVSC, &client.DeleteOptions{
				GracePeriodSeconds: vVSC.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pVSC.UID)),
			})
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		updatedPv := s.translateUpdate(vVSC, pVSC)
		if updatedPv != nil {
			log.Infof("update physical VolumeSnapshotContent %s, because spec or annotations have changed", updatedPv.Name)
			err := s.localClient.Update(ctx, updatedPv)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) shouldSync(ctx context.Context, pObj *volumesnapshotv1.VolumeSnapshotContent) (bool, *volumesnapshotv1.VolumeSnapshot, error) {
	if pObj.Spec.VolumeSnapshotRef.Namespace != s.targetNamespace {
		return false, nil, nil
	}

	vVS := &volumesnapshotv1.VolumeSnapshot{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vVS, constants.IndexByPhysicalName, pObj.Spec.VolumeSnapshotRef.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.IsManagedCluster(s.targetNamespace, pObj) {
			return true, nil, nil
		}
		return false, nil, nil
	}

	return true, vVS, nil
}

func (s *syncer) IsManaged(pObj client.Object) (bool, error) {
	pVSC, ok := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok {
		return false, nil
	}

	sync, _, err := s.shouldSync(context.TODO(), pVSC)
	if err != nil {
		return false, nil
	}

	return sync, nil
}

func (s *syncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translateVolumeSnapshotContentName(s.targetNamespace, req.Name, vObj)}
}

func (s *syncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
		return types.NamespacedName{
			Name: pAnnotations[translate.NameAnnotation],
		}
	}

	vObj := &volumesnapshotv1.VolumeSnapshotContent{}
	err := clienthelper.GetByIndex(context.Background(), s.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetName())
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: pObj.GetName()}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translateVolumeSnapshotContentName(physicalNamespace, name string, vObj runtime.Object) string {
	if vObj == nil {
		return name
	}

	vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok || vVSC.Annotations == nil || vVSC.Annotations[HostClusterVSCAnnotation] == "" {
		return translate.PhysicalNameClusterScoped(name, physicalNamespace)
	}

	return vVSC.Annotations[HostClusterVSCAnnotation]
}
