package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
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

type backwardController struct {
	target Syncer

	targetNamespace string
	log             loghelper.Logger
	localClient     client.Client
	virtualClient   client.Client
	scheme          *runtime.Scheme
}

func (r *backwardController) GarbageCollect(queue workqueue.RateLimitingInterface) error {
	ctx := context.Background()

	// list all physical objects first
	pList := r.target.NewList()
	err := r.localClient.List(ctx, pList, client.InNamespace(r.targetNamespace))
	if err != nil {
		return err
	}

	// check if physical object exists
	items, err := meta.ExtractList(pList)
	if err != nil {
		return err
	}
	for _, pObj := range items {
		if translate.IsManaged(pObj) == false {
			continue
		}

		vObj := r.target.New()
		pAccessor, _ := meta.Accessor(pObj)
		err = clienthelper.GetByIndex(ctx, r.virtualClient, vObj, r.scheme, constants.IndexByVName, pAccessor.GetName())
		if kerrors.IsNotFound(err) {
			r.log.Debugf("resync physical object %s/%s, because virtual object is missing", pAccessor.GetNamespace(), pAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pAccessor.GetName(),
					Namespace: pAccessor.GetNamespace(),
				},
			})
			continue
		} else if err != nil {
			r.log.Infof("cannot get physical object %s/%s: %v", r.targetNamespace, pAccessor.GetName(), err)
			continue
		}

		updateNeeded, err := r.target.BackwardUpdateNeeded(pObj.(client.Object), vObj)
		if err != nil {
			r.log.Infof("error in update needed for physical object %s/%s: %v", pAccessor.GetNamespace(), pAccessor.GetName(), err)
			continue
		}

		if updateNeeded {
			r.log.Debugf("resync physical object %s/%s", pAccessor.GetNamespace(), pAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pAccessor.GetName(),
					Namespace: pAccessor.GetNamespace(),
				},
			})
		}
	}

	return nil
}

func (r *backwardController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// check if we should skip reconcile
	lifecycle, ok := r.target.(BackwardLifecycle)
	if ok {
		skip, err := lifecycle.BackwardStart(ctx, req)
		defer lifecycle.BackwardEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get physical object
	pObj := r.target.New()
	err := r.localClient.Get(ctx, req.NamespacedName, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			r.log.Infof("error retrieving physical object %s/%s: %v", req.Namespace, req.Name, err)
		}

		return ctrl.Result{}, nil
	}

	if !translate.IsManaged(pObj) {
		return ctrl.Result{}, nil
	}

	vObj := r.target.New()
	err = clienthelper.GetByIndex(ctx, r.virtualClient, vObj, r.scheme, constants.IndexByVName, req.Name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			if backwardDeleter, ok := r.target.(BackwardDelete); ok {
				return backwardDeleter.BackwardDelete(ctx, pObj, r.log)
			}

			return DeleteObject(ctx, r.localClient, pObj, r.log)
		}

		return ctrl.Result{}, err
	}

	return r.target.BackwardUpdate(ctx, pObj, vObj, r.log)
}
