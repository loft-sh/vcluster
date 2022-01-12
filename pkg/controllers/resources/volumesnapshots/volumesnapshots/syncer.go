package volumesnapshots

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &volumesnapshotv1.VolumeSnapshot{})
	if err != nil {
		return err
	}
	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	return generic.RegisterSyncer(ctx, "volumesnapshot", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &volumesnapshotv1.VolumeSnapshot{}),

		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "volumesnapshot-syncer"}), "volumesnapshot"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace),

		volumeSnapshotContentNameTranslator: volumesnapshotcontents.NewVolumeSnapshotContentTranslator(ctx.Options.TargetNamespace),
	})
}

type syncer struct {
	generic.Translator

	targetNamespace string
	virtualClient   client.Client
	localClient     client.Client

	creator    *generic.GenericCreator
	translator translate.Translator

	volumeSnapshotContentNameTranslator translate.PhysicalNameTranslator
}

func (s *syncer) New() client.Object {
	return &volumesnapshotv1.VolumeSnapshot{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vVS := vObj.(*volumesnapshotv1.VolumeSnapshot)
	if vVS.DeletionTimestamp != nil {
		// delete volume snapshot immediately
		if len(vObj.GetFinalizers()) > 0 || (vObj.GetDeletionGracePeriodSeconds() != nil && *vObj.GetDeletionGracePeriodSeconds() > 0) {
			vObj.SetFinalizers([]string{})
			vObj.SetDeletionGracePeriodSeconds(&zero)
			return ctrl.Result{}, s.virtualClient.Update(ctx, vObj)
		}
		return ctrl.Result{}, nil
	}

	pObj, err := s.translate(ctx, vVS)
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vVS := vObj.(*volumesnapshotv1.VolumeSnapshot)
	pVS := pObj.(*volumesnapshotv1.VolumeSnapshot)

	if pVS.DeletionTimestamp != nil {
		if vVS.DeletionTimestamp == nil {
			log.Infof("delete virtual volume snapshot %s/%s, because the physical volume snapshot is being deleted", vVS.Namespace, vVS.Name)
			err := s.virtualClient.Delete(ctx, vVS, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if *vVS.DeletionGracePeriodSeconds != *pVS.DeletionGracePeriodSeconds {
			log.Infof("delete virtual volume snapshot %s/%s with grace period seconds %v", vVS.Namespace, vVS.Name, *pVS.DeletionGracePeriodSeconds)
			err := s.virtualClient.Delete(ctx, vVS, &client.DeleteOptions{GracePeriodSeconds: pVS.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vVS.UID))})
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
			log.Infof("update finalizers of the virtual VolumeSnapshot %s, because finalizers on the physical resource changed", vVS.Name)
			err := s.virtualClient.Update(ctx, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		if !equality.Semantic.DeepEqual(vVS.Status, pVS.Status) {
			vVS.Status = pVS.Status.DeepCopy()
			log.Infof("update virtual VolumeSnapshot %s, because status has changed", vVS.Name)
			err := s.virtualClient.Status().Update(ctx, vVS)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil

	} else if vVS.DeletionTimestamp != nil {
		if pVS.DeletionTimestamp == nil {
			log.Infof("delete physical volume snapshot %s/%s, because virtual volume snapshot is being deleted", pVS.Namespace, pVS.Name)
			return ctrl.Result{}, s.localClient.Delete(ctx, pVS, &client.DeleteOptions{
				GracePeriodSeconds: vVS.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pVS.UID)),
			})
		}
		return ctrl.Result{}, nil
	}

	// check backwards update
	updated := s.translateUpdateBackwards(pVS, vVS)
	if updated != nil {
		log.Infof("update virtual volume snapshot %s/%s, because the spec has changed", vVS.Namespace, vVS.Name)
		err := s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// check backwards status
	if !equality.Semantic.DeepEqual(vVS.Status, pVS.Status) {
		vVS.Status = pVS.Status.DeepCopy()
		log.Infof("update virtual volume snapshot %s/%s, because the status has changed", vVS.Namespace, vVS.Name)
		err := s.virtualClient.Status().Update(ctx, vVS)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// forward update
	return s.creator.Update(ctx, vVS, s.translateUpdate(pVS, vVS), log)
}
