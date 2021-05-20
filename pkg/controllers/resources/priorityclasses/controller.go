package priorityclasses

import (
	"context"
	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/garbagecollect"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func registerForwardSyncer(ctx *controllercontext.ControllerContext, syncer generic.Syncer, name string, isManaged func(obj runtime.Object) bool, physicalName func(name string) string) error {
	forwardController := &forwardController{
		isManaged:     isManaged,
		physicalName:  physicalName,
		target:        syncer,
		synced:        ctx.CacheSynced,
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
		scheme:        ctx.LocalManager.GetScheme(),
		log:           loghelper.New(name + "-forward"),
	}
	b := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-forward").
		Watches(garbagecollect.NewGarbageCollectSource(forwardController, ctx.StopChan, forwardController.log), nil).
		For(syncer.New())
	return b.Complete(forwardController)
}

type forwardController struct {
	synced func()

	isManaged    func(obj runtime.Object) bool
	physicalName func(name string) string

	target        generic.Syncer
	log           loghelper.Logger
	localClient   client.Client
	virtualClient client.Client
	scheme        *runtime.Scheme
}

func (r *forwardController) GarbageCollect(queue workqueue.RateLimitingInterface) error {
	ctx := context.Background()

	// list all virtual objects first
	vList := r.target.NewList()
	err := r.virtualClient.List(ctx, vList)
	if err != nil {
		return err
	}

	// check if physical object exists
	vItems, err := meta.ExtractList(vList)
	if err != nil {
		return err
	}
	for _, vObj := range vItems {
		vAccessor, _ := meta.Accessor(vObj)
		pObj := r.target.New()
		err = r.localClient.Get(ctx, types.NamespacedName{
			Name: r.physicalName(vAccessor.GetName()),
		}, pObj)
		if kerrors.IsNotFound(err) {
			fc, ok := r.target.(generic.ForwardCreate)
			if ok {
				createNeeded, err := fc.ForwardCreateNeeded(vObj.(client.Object))
				if err != nil {
					r.log.Infof("error in create needed for virtual object %s: %v", vAccessor.GetName(), err)
					continue
				} else if createNeeded == false {
					continue
				}
			}

			r.log.Debugf("resync virtual object %s, because physical object was not found", vAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: vAccessor.GetName(),
				},
			})
			continue
		} else if err != nil {
			r.log.Infof("cannot get physical object %s: %v", translate.PhysicalName(vAccessor.GetName(), vAccessor.GetNamespace()), err)
			continue
		}

		updateNeeded, err := r.target.ForwardUpdateNeeded(pObj, vObj.(client.Object))
		if err != nil {
			r.log.Infof("error in update needed for virtual object %s/%s: %v", vAccessor.GetNamespace(), vAccessor.GetName(), err)
			continue
		}

		if updateNeeded {
			r.log.Debugf("resync virtual object %s/%s", vAccessor.GetNamespace(), vAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      vAccessor.GetName(),
					Namespace: vAccessor.GetNamespace(),
				},
			})
		}
	}

	// list all physical objects first
	pList := r.target.NewList()
	err = r.localClient.List(ctx, pList)
	if err != nil {
		return err
	}

	// check if physical object exists
	items, err := meta.ExtractList(pList)
	if err != nil {
		return err
	}
	for _, pObj := range items {
		if r.isManaged(pObj) == false {
			continue
		}

		vObj := r.target.New()
		pAccessor, _ := meta.Accessor(pObj)
		err = clienthelper.GetByIndex(ctx, r.virtualClient, vObj, r.scheme, constants.IndexByVName, pAccessor.GetName())
		if kerrors.IsNotFound(err) {
			r.log.Debugf("resync physical object %s, because virtual object is missing", pAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: pAccessor.GetName(),
				},
			})
			continue
		}
	}

	return nil
}

func (r *forwardController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// make sure the caches are synced
	r.synced()

	// get virtual object
	vObj := r.target.New()
	vExists := true
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		vExists = false
	}

	// get physical object
	pObj := r.target.New()
	pExists := true
	err = r.localClient.Get(ctx, types.NamespacedName{
		Name: r.physicalName(req.Name),
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		pExists = false
	}

	if vExists && !pExists {
		return r.target.ForwardCreate(ctx, vObj, r.log)
	} else if vExists && pExists {
		return r.target.ForwardUpdate(ctx, pObj, vObj, r.log)
	} else if !vExists && pExists {
		if !r.isManaged(pObj) {
			return ctrl.Result{}, nil
		}

		return generic.DeleteObject(ctx, r.localClient, pObj, r.log)
	}

	return ctrl.Result{}, nil
}
