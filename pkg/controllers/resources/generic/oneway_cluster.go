package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type oneWayClusterController struct {
	target    OneWayClusterSyncer
	lifecycle BackwardLifecycle

	log           loghelper.Logger
	localClient   client.Client
	virtualClient client.Client
}

func (r *oneWayClusterController) GarbageCollect(queue workqueue.RateLimitingInterface) error {
	ctx := context.Background()

	// list all virtual objects first
	vList := r.target.NewList()
	err := r.virtualClient.List(ctx, vList)
	if err != nil {
		return err
	}

	// extract
	vItems, err := meta.ExtractList(vList)
	if err != nil {
		return err
	}
	for _, vObj := range vItems {
		// get physical object
		pObj := r.target.New()
		vAccessor, _ := meta.Accessor(vObj)
		err = r.localClient.Get(ctx, types.NamespacedName{Name: vAccessor.GetName()}, pObj)
		if err != nil {
			if kerrors.IsNotFound(err) {
				// requeue object
				r.log.Debugf("resync physical object %s", vAccessor.GetName())
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: vAccessor.GetName(),
					},
				})
			} else {
				r.log.Infof("error retrieving physical object %s: %v", vAccessor.GetName(), err)
			}

			continue
		}

		pAccessor, _ := meta.Accessor(pObj)
		updateNeeded, err := r.target.BackwardUpdateNeeded(pObj, vObj.(client.Object))
		if err != nil {
			r.log.Infof("error in update needed for physical object %s: %v", pAccessor.GetName(), err)
			continue
		}

		if updateNeeded {
			r.log.Debugf("resync physical object %s", pAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: pAccessor.GetName(),
				},
			})
		}
	}

	// list all physical objects
	pList := r.target.NewList()
	err = r.localClient.List(ctx, pList)
	if err != nil {
		return err
	}

	// extract
	pItems, err := meta.ExtractList(pList)
	if err != nil {
		return err
	}
	for _, pObj := range pItems {
		pAccessor, _ := meta.Accessor(pObj)
		vObj := r.target.New()
		err = r.virtualClient.Get(ctx, types.NamespacedName{Name: pAccessor.GetName()}, vObj)
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return err
			}

			needed, err := r.target.BackwardCreateNeeded(pObj.(client.Object))
			if err != nil {
				return err
			}

			if needed {
				// requeue object
				r.log.Debugf("resync physical object %s", pAccessor.GetName())
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: pAccessor.GetName(),
					},
				})
			}
		}
	}

	return nil
}

func (r *oneWayClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// check if we should skip reconcile
	if r.lifecycle != nil {
		skip, err := r.lifecycle.BackwardStart(ctx, req)
		defer r.lifecycle.BackwardEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get physical object
	pObj := r.target.New()
	err := r.localClient.Get(ctx, req.NamespacedName, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		// got deleted
		vObj := r.target.New()
		err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}

		r.log.Debugf("delete virtual object %s, because physical got deleted", req.Name)
		return ctrl.Result{}, r.virtualClient.Delete(ctx, vObj)
	}

	// check if there is a virtual object for this
	vObj := r.target.New()
	err = r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.target.BackwardCreate(ctx, pObj, r.log)
		}

		return ctrl.Result{}, err
	}

	return r.target.BackwardUpdate(ctx, pObj, vObj, r.log)
}
