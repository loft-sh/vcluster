package syncer

import (
	"context"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type enqueueFunc func(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface, isDelete bool)

func newEventHandler(enqueue enqueueFunc) handler.EventHandler {
	return &eventHandler{enqueue: enqueue}
}

type eventHandler struct {
	enqueue enqueueFunc
}

// Create is called in response to an create event - e.g. Pod Creation.
func (r *eventHandler) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.enqueue(ctx, evt.Object, q, false)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (r *eventHandler) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.enqueue(ctx, evt.ObjectNew, q, false)
}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (r *eventHandler) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.enqueue(ctx, evt.Object, q, true)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func (r *eventHandler) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	r.enqueue(ctx, evt.Object, q, false)
}
