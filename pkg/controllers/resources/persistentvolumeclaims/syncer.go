package persistentvolumeclaims

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/persistentvolumes"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	skipPVTranslationAnnotation = "vcluster.loft.sh/skip-translate"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &corev1.PersistentVolumeClaim{})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	return generic.RegisterSyncer(ctx, "persistentvolumeclaim", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.PersistentVolumeClaim{}),

		useFakePersistentVolumes:     !ctx.Controllers["persistentvolumes"],
		sharedPersistentVolumesMutex: ctx.LockFactory.GetLock("persistent-volumes-controller"),

		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "persistentvolumeclaim-syncer"}), "persistent volume claim"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace, bindCompletedAnnotation, boundByControllerAnnotation, storageProvisionerAnnotation),
	})
}

type syncer struct {
	generic.Translator

	useFakePersistentVolumes     bool
	sharedPersistentVolumesMutex sync.Locker

	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client

	creator    *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &corev1.PersistentVolumeClaim{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	if vPvc.DeletionTimestamp != nil {
		// delete pvc immediately
		log.Infof("delete virtual persistent volume claim %s/%s immediately, because it is being deleted & there is no physical persistent volume claim", vPvc.Namespace, vPvc.Name)
		err := s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	pObj, err := s.translate(s.targetNamespace, vPvc)
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPvc := vObj.(*corev1.PersistentVolumeClaim)
	pPvc := pObj.(*corev1.PersistentVolumeClaim)

	// if pvs are deleted check the corresponding pvc is deleted as well
	if pPvc.DeletionTimestamp != nil {
		if vPvc.DeletionTimestamp == nil {
			log.Infof("delete virtual persistent volume claim %s/%s, because the physical persistent volume claim is being deleted", vPvc.Namespace, vPvc.Name)
			if err := s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPvc.DeletionGracePeriodSeconds != *pPvc.DeletionGracePeriodSeconds {
			log.Infof("delete virtual persistent volume claim %s/%s with grace period seconds %v", vPvc.Namespace, vPvc.Name, *pPvc.DeletionGracePeriodSeconds)
			if err := s.virtualClient.Delete(ctx, vPvc, &client.DeleteOptions{GracePeriodSeconds: pPvc.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPvc.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if vPvc.DeletionTimestamp != nil {
		log.Infof("delete physical persistent volume claim %s/%s, because virtual persistent volume claim is being deleted", pPvc.Namespace, pPvc.Name)
		err := s.localClient.Delete(ctx, pPvc, &client.DeleteOptions{
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
		err := s.ensurePersistentVolume(ctx, pPvc, vPvc, log)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// check backwards update
	updated := s.translateUpdateBackwards(pPvc, vPvc)
	if updated != nil {
		log.Infof("update virtual persistent volume claim %s/%s, because the spec has changed", vPvc.Namespace, vPvc.Name)
		err := s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	// check backwards status
	if !equality.Semantic.DeepEqual(vPvc.Status, pPvc.Status) {
		vPvc.Status = *pPvc.Status.DeepCopy()
		log.Infof("update virtual persistent volume claim %s/%s, because the status has changed", vPvc.Namespace, vPvc.Name)
		err := s.virtualClient.Status().Update(ctx, vPvc)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	// forward update
	return s.creator.Update(ctx, vPvc, s.translateUpdate(pPvc, vPvc), log)
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
			log.Infof("create virtual fake pv %s, because pvc %s/%s uses it and it is not available in virtual cluster", pObj.Spec.VolumeName, vObj.Namespace, vObj.Name)

			// create fake persistent volume
			err = persistentvolumes.CreateFakePersistentVolume(ctx, s.virtualClient, types.NamespacedName{Name: pObj.Spec.VolumeName}, vObj)
			if err != nil {
				log.Infof("error creating virtual fake persistent volume %s: %v", pObj.Spec.VolumeName, err)
				return err
			}
		}
	}

	if pObj.Spec.VolumeName != "" && vObj.Spec.VolumeName != pObj.Spec.VolumeName {
		newVolumeName := pObj.Spec.VolumeName
		if s.useFakePersistentVolumes == false {
			vObj := &corev1.PersistentVolume{}
			err = clienthelper.GetByIndex(ctx, s.virtualClient, vObj, constants.IndexByPhysicalName, pObj.Spec.VolumeName)
			if err != nil {
				log.Infof("error retrieving virtual persistent volume %s: %v", pObj.Spec.VolumeName, err)
				return err
			}

			newVolumeName = vObj.Name
		}

		if newVolumeName != vObj.Spec.VolumeName {
			log.Infof("update virtual pvc %s/%s volume name to %s", vObj.Namespace, vObj.Name, newVolumeName)

			vObj.Spec.VolumeName = newVolumeName
			err = s.virtualClient.Update(ctx, vObj)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
