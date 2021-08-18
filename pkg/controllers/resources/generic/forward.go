package generic

import (
	"context"
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

type forwardController struct {
	target          Syncer
	targetNamespace string
	log             loghelper.Logger
	localClient     client.Client
	virtualClient   client.Client
	scheme          *runtime.Scheme
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
			Namespace: r.targetNamespace,
			Name:      translate.PhysicalName(vAccessor.GetName(), vAccessor.GetNamespace()),
		}, pObj)
		if kerrors.IsNotFound(err) {
			fc, ok := r.target.(ForwardCreate)
			if ok {
				createNeeded, err := fc.ForwardCreateNeeded(vObj.(client.Object))
				if err != nil {
					r.log.Infof("error in create needed for virtual object %s/%s: %v", vAccessor.GetNamespace(), vAccessor.GetName(), err)
					continue
				} else if createNeeded == false {
					continue
				}
			}

			r.log.Debugf("resync virtual object %s/%s, because physical object was not found", vAccessor.GetNamespace(), vAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      vAccessor.GetName(),
					Namespace: vAccessor.GetNamespace(),
				},
			})
			continue
		} else if err != nil {
			r.log.Infof("cannot get physical object %s/%s: %v", r.targetNamespace, translate.PhysicalName(vAccessor.GetName(), vAccessor.GetNamespace()), err)
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

	return nil
}

func (r *forwardController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// check if we should skip reconcile
	lifecycle, ok := r.target.(ForwardLifecycle)
	if ok {
		skip, err := lifecycle.ForwardStart(ctx, req)
		defer lifecycle.ForwardEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

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
		Namespace: r.targetNamespace,
		Name:      translate.PhysicalName(req.Name, req.Namespace),
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
		if !translate.IsManaged(pObj) {
			return ctrl.Result{}, nil
		}

		return DeleteObject(ctx, r.localClient, pObj, r.log)
	}

	return ctrl.Result{}, nil
}

func DeleteObject(ctx context.Context, localClient client.Client, pObj runtime.Object, log loghelper.Logger) (ctrl.Result, error) {
	accessor, err := meta.Accessor(pObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("delete physical %s/%s, because virtual object was deleted", accessor.GetNamespace(), accessor.GetName())
	err = localClient.Delete(ctx, pObj.(client.Object))
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Infof("error deleting physical object %s/%s in physical cluster: %v", accessor.GetNamespace(), accessor.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
