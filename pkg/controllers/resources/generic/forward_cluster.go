package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type forwardClusterController struct {
	synced func()

	target        ClusterSyncer
	log           loghelper.Logger
	localClient   client.Client
	virtualClient client.Client
	scheme        *runtime.Scheme
}

func (r *forwardClusterController) GarbageCollect(queue workqueue.RateLimitingInterface) error {
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
			Name: vAccessor.GetName(),
		}, pObj)
		if kerrors.IsNotFound(err) {
			// we ignore this case as we only update cluster resources on host, but never create them ourselves
			continue
		} else if err != nil {
			r.log.Infof("cannot get physical object %s: %v", vAccessor.GetName(), err)
			continue
		}

		updateNeeded, err := r.target.ForwardUpdateNeeded(pObj, vObj.(client.Object))
		if err != nil {
			r.log.Infof("error in update needed for virtual object %s: %v", vAccessor.GetName(), err)
			continue
		}

		if updateNeeded {
			r.log.Debugf("resync virtual object %s", vAccessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: vAccessor.GetName(),
				},
			})
		}
	}

	return nil
}

func (r *forwardClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// make sure the caches are synced
	r.synced()

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
	err = r.localClient.Get(ctx, req.NamespacedName, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		pExists = false
	}

	if vExists && pExists {
		return r.target.ForwardUpdate(ctx, pObj, vObj, r.log)
	}

	return ctrl.Result{}, nil
}
