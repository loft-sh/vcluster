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

type fakeSyncer struct {
	synced func()
	target FakeSyncer

	virtualClient client.Client
	log           loghelper.Logger
}

func (r *fakeSyncer) GarbageCollect(queue workqueue.RateLimitingInterface) error {
	ctx := context.Background()
	list := r.target.NewList()
	err := r.virtualClient.List(ctx, list)
	if err != nil {
		return err
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	// delete items that are not needed anymore
	for _, item := range items {
		accessor, _ := meta.Accessor(item)
		shouldDelete, err := r.target.DeleteNeeded(ctx, item.(client.Object))
		if err != nil {
			r.log.Infof("cannot determine if still needed %s: %v", accessor.GetNamespace(), accessor.GetName(), err)
			continue
		} else if shouldDelete {
			r.log.Debugf("resync object %s", accessor.GetName())
			queue.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      accessor.GetName(),
					Namespace: accessor.GetNamespace(),
				},
			})
		}
	}

	// create items that are needed but not there
	list = r.target.DependantObjectList()
	err = r.virtualClient.List(ctx, list)
	if err != nil {
		return err
	}
	items, err = meta.ExtractList(list)
	if err != nil {
		return err
	}

	for _, item := range items {
		accessor, _ := meta.Accessor(item)
		name, err := r.target.NameFromDependantObject(ctx, item.(client.Object))
		if err != nil {
			r.log.Infof("cannot determine name of object %s/%s: %v", accessor.GetNamespace(), accessor.GetName(), err)
			continue
		} else if name.Name == "" {
			continue
		}

		item := r.target.New()
		err = r.virtualClient.Get(ctx, name, item)
		if err == nil {
			continue
		} else if kerrors.IsNotFound(err) == false {
			r.log.Infof("cannot get object %s: %v", name.Name, err)
			continue
		}

		r.log.Debugf("resync object %s, because it is missing, but needed", accessor.GetName())
		queue.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      accessor.GetName(),
				Namespace: accessor.GetNamespace(),
			},
		})
	}

	return nil
}

func (r *fakeSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// make sure the caches are synced
	r.synced()

	// check if we should skip reconcile
	skip, err := r.target.ReconcileStart(ctx, req)
	defer r.target.ReconcileEnd()
	if skip || err != nil {
		return ctrl.Result{}, err
	}

	// get virtual object
	vObj := r.target.New()
	err = r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		needed, err := r.target.CreateNeeded(ctx, req.NamespacedName)
		if err != nil {
			return ctrl.Result{}, err
		} else if needed {
			err = r.target.Create(ctx, req.NamespacedName)
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, nil
	}

	// check if object is still needed
	shouldDelete, err := r.target.DeleteNeeded(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if shouldDelete {
		accessor, _ := meta.Accessor(vObj)
		r.log.Debugf("garbage collect virtual %s, because it is not needed any longer", accessor.GetName())
		err = r.target.Delete(ctx, vObj)
		if err != nil {
			r.log.Infof("cannot delete virtual %s: %v", accessor.GetName(), err)
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
