package translator

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGenericTranslator(ctx *synccontext.RegisterContext, name string, obj client.Object, mapper synccontext.Mapper) syncertypes.GenericTranslator {
	return &genericTranslator{
		Mapper: mapper,

		name: name,

		obj: obj,

		eventRecorder: ctx.VirtualManager.GetEventRecorder(name + "-syncer"),
	}
}

type genericTranslator struct {
	synccontext.Mapper

	name string

	obj client.Object

	eventRecorder events.EventRecorder
}

func (n *genericTranslator) EventRecorder() events.EventRecorder {
	return n.eventRecorder
}

func (n *genericTranslator) Name() string {
	return n.name
}

func (n *genericTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}
