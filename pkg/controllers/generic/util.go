package generic

import (
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGenericCreator(localClient client.Client, eventRecorder record.EventRecorder, name string) *GenericCreator {
	return &GenericCreator{}
}

type GenericCreator struct {
}
