package synccontext

import "sigs.k8s.io/controller-runtime/pkg/client"

type SyncEventType string

const (
	SyncEventTypeUnknown       SyncEventType = ""
	SyncEventTypeDelete        SyncEventType = "Delete"
	SyncEventTypePendingDelete SyncEventType = "PendingDelete"
)

type SyncEventSource string

const (
	SyncEventSourceHost    SyncEventSource = "Host"
	SyncEventSourceVirtual SyncEventSource = "Virtual"
)

func NewSyncToHostEvent[T client.Object](vObj T) *SyncToHostEvent[T] {
	return &SyncToHostEvent[T]{
		Source: SyncEventSourceVirtual,

		Virtual: vObj,
	}
}

func NewSyncToVirtualEvent[T client.Object](pObj T) *SyncToVirtualEvent[T] {
	return &SyncToVirtualEvent[T]{
		Source: SyncEventSourceVirtual,

		Host: pObj,
	}
}

func NewSyncEvent[T client.Object](pObj, vObj T) *SyncEvent[T] {
	return &SyncEvent[T]{
		Source: SyncEventSourceVirtual,

		Host:    pObj,
		Virtual: vObj,
	}
}

func NewSyncEventWithSource[T client.Object](pObj, vObj T, source SyncEventSource) *SyncEvent[T] {
	return &SyncEvent[T]{
		Source: source,

		Host:    pObj,
		Virtual: vObj,
	}
}

type SyncEvent[T client.Object] struct {
	Type   SyncEventType
	Source SyncEventSource

	Host    T
	Virtual T
}

func (s *SyncEvent[T]) SourceObject() T {
	if s.Source == SyncEventSourceHost {
		return s.Host
	}
	return s.Virtual
}

func (s *SyncEvent[T]) TargetObject() T {
	if s.Source == SyncEventSourceHost {
		return s.Virtual
	}
	return s.Host
}

func (s *SyncEvent[T]) IsDelete() bool {
	return s.Type == SyncEventTypeDelete
}

type SyncToHostEvent[T client.Object] struct {
	Type   SyncEventType
	Source SyncEventSource

	Virtual T
}

func (s *SyncToHostEvent[T]) IsDelete() bool {
	return s.Type == SyncEventTypeDelete
}

type SyncToVirtualEvent[T client.Object] struct {
	Type   SyncEventType
	Source SyncEventSource

	Host T
}

func (s *SyncToVirtualEvent[T]) IsDelete() bool {
	return s.Type == SyncEventTypeDelete
}
