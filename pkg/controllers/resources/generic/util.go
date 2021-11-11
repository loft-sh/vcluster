package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGenericCreator(localClient client.Client, eventRecorder record.EventRecorder, name string) *GenericCreator {
	return &GenericCreator{
		localClient:   localClient,
		eventRecorder: eventRecorder,
		name: name,
	}
}

type GenericCreator struct {
	localClient   client.Client
	eventRecorder record.EventRecorder
	
	name string
}

func (g *GenericCreator) Create(ctx context.Context, vObj, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	log.Infof("create physical %s %s/%s", g.name, pObj.GetNamespace(), pObj.GetName())
	err := g.localClient.Create(ctx, pObj)
	if err != nil {
		log.Infof("error syncing %s %s/%s to physical cluster: %v", g.name, vObj.GetNamespace(), vObj.GetName(), err)
		g.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (g *GenericCreator) Update(ctx context.Context, vObj, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	// this is needed because of interface nil check
	if !(pObj == nil || (reflect.ValueOf(pObj).Kind() == reflect.Ptr && reflect.ValueOf(pObj).IsNil())) {
		log.Infof("updating physical %s/%s, because virtual %s have changed", pObj.GetNamespace(), pObj.GetName(), g.name)
		err := g.localClient.Update(ctx, pObj)
		if err != nil {
			g.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
