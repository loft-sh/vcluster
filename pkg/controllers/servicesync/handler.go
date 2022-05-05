package servicesync

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type serviceHandler struct {
	Mapping map[string]types.NamespacedName
}

// Create implements EventHandler
func (e *serviceHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.handleEvent(evt.Object, q)
}

// Update implements EventHandler
func (e *serviceHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.handleEvent(evt.ObjectNew, q)
}

// Delete implements EventHandler
func (e *serviceHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.handleEvent(evt.Object, q)
}

// Generic implements EventHandler
func (e *serviceHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.handleEvent(evt.Object, q)
}

func (e *serviceHandler) handleEvent(meta metav1.Object, q workqueue.RateLimitingInterface) {
	if meta == nil {
		return
	}

	from, ok := e.Mapping[meta.GetNamespace()+"/"+meta.GetName()]
	if !ok {
		return
	}

	q.Add(reconcile.Request{NamespacedName: from})
}
