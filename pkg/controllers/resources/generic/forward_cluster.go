package generic

import (
	"context"
	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/garbagecollect"
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

func registerForwardClusterSyncer(ctx *controllercontext.ControllerContext, syncer TwoWayClusterSyncer, name string) error {
	forwardClusterController := &forwardClusterController{
		syncer:        syncer,
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
		scheme:        ctx.LocalManager.GetScheme(),
		log:           loghelper.New(name + "-forward"),
	}
	b := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-forward").
		Watches(garbagecollect.NewGarbageCollectSource(forwardClusterController, ctx.StopChan, forwardClusterController.log), nil).
		For(syncer.New())
	return b.Complete(forwardClusterController)
}

type forwardClusterController struct {
	syncer        TwoWayClusterSyncer
	log           loghelper.Logger
	localClient   client.Client
	virtualClient client.Client
	scheme        *runtime.Scheme
}

func (r *forwardClusterController) GarbageCollect(queue workqueue.RateLimitingInterface) error {
	ctx := context.Background()

	// list all virtual objects first
	vList := r.syncer.NewList()
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
		pObj := r.syncer.New()
		err = r.localClient.Get(ctx, types.NamespacedName{
			Name: r.syncer.PhysicalName(vAccessor.GetName(), vObj),
		}, pObj)
		if kerrors.IsNotFound(err) {
			fc, ok := r.syncer.(ForwardCreate)
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

		updateNeeded, err := r.syncer.ForwardUpdateNeeded(pObj, vObj.(client.Object))
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

	return nil
}

func (r *forwardClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// check if we should skip reconcile
	lifecycle, ok := r.syncer.(ForwardLifecycle)
	if ok {
		skip, err := lifecycle.ForwardStart(ctx, req)
		defer lifecycle.ForwardEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get virtual object
	vObj := r.syncer.New()
	vExists := true
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		vExists = false
	}

	// get physical object
	pObj := r.syncer.New()
	pExists := true
	err = r.localClient.Get(ctx, types.NamespacedName{
		Name: r.syncer.PhysicalName(req.Name, vObj),
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		pExists = false
	}

	if vExists && !pExists {
		return r.syncer.ForwardCreate(ctx, vObj, r.log)
	} else if vExists && pExists {
		return r.syncer.ForwardUpdate(ctx, pObj, vObj, r.log)
	} else if !vExists && pExists {
		if !r.syncer.IsManaged(pObj) {
			return ctrl.Result{}, nil
		}

		return DeleteObject(ctx, r.localClient, pObj, r.log)
	}

	return ctrl.Result{}, nil
}
