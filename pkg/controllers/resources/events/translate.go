package events

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

func (s *eventSyncer) translateEvent(ctx *synccontext.SyncContext, pEvent, vEvent *corev1.Event) error {
	// retrieve involved object
	involvedObject, err := resources.GetInvolvedObject(ctx, pEvent)
	if err != nil {
		return err
	}
	tempEvent := pEvent.DeepCopy()

	// set the correct involved object meta
	tempEvent.InvolvedObject.Namespace = involvedObject.GetNamespace()
	tempEvent.InvolvedObject.Name = involvedObject.GetName()
	tempEvent.InvolvedObject.UID = involvedObject.GetUID()
	tempEvent.InvolvedObject.ResourceVersion = involvedObject.GetResourceVersion()

	// rewrite name
	namespace := involvedObject.GetNamespace()
	name := hostEventNameToVirtual(pEvent.Name, pEvent.InvolvedObject.Name, involvedObject.GetName())

	// we replace namespace/name & name in messages so that it seems correct
	tempEvent.Message = strings.ReplaceAll(tempEvent.Message, pEvent.InvolvedObject.Namespace+"/"+pEvent.InvolvedObject.Name, tempEvent.InvolvedObject.Namespace+"/"+tempEvent.InvolvedObject.Name)
	tempEvent.Message = strings.ReplaceAll(tempEvent.Message, pEvent.InvolvedObject.Name, tempEvent.InvolvedObject.Name)

	// keep the metadata from the virtual object
	translate.ResetObjectMetadata(tempEvent)
	tempEvent.ObjectMeta = vEvent.ObjectMeta
	tempEvent.TypeMeta = vEvent.TypeMeta

	tempEvent.DeepCopyInto(vEvent)
	vEvent.Namespace = namespace
	vEvent.Name = name
	return nil
}

func hostEventNameToVirtual(hostName string, hostInvolvedObjectName, virtualInvolvedObjectName string) string {
	// replace name of object
	if strings.HasPrefix(hostName, hostInvolvedObjectName) {
		hostName = strings.Replace(hostName, hostInvolvedObjectName, virtualInvolvedObjectName, 1)
	}

	return hostName
}
