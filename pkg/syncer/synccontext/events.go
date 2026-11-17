package synccontext

import "sigs.k8s.io/controller-runtime/pkg/client"

type SyncDirection string

const (
	SyncVirtualToHost SyncDirection = "VirtualToHost"
	SyncHostToVirtual SyncDirection = "HostToVirtual"
)

func NewSyncToHostEvent[T client.Object](vObj T) *SyncToHostEvent[T] {
	return &SyncToHostEvent[T]{
		Virtual: vObj,
	}
}

func NewSyncToVirtualEvent[T client.Object](pObj T) *SyncToVirtualEvent[T] {
	return &SyncToVirtualEvent[T]{
		Host: pObj,
	}
}

func NewSyncEvent[T client.Object](pObj, vObj T) *SyncEvent[T] {
	return &SyncEvent[T]{
		Host:    pObj,
		Virtual: vObj,
	}
}

func NewSyncEventWithOld[T client.Object](pObjOld, pObj, vObjOld, vObj T) *SyncEvent[T] {
	return &SyncEvent[T]{
		HostOld: pObjOld,
		Host:    pObj,

		VirtualOld: vObjOld,
		Virtual:    vObj,
	}
}

type SyncEvent[T client.Object] struct {
	HostOld T
	Host    T

	VirtualOld T
	Virtual    T
}

type SyncToHostEvent[T client.Object] struct {
	HostOld T

	Virtual T
}

type SyncToVirtualEvent[T client.Object] struct {
	VirtualOld T

	Host T
}
