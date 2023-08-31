package telemetry

import (
	"sync"

	"github.com/loft-sh/vcluster/pkg/telemetry/types"
)

func newEventBuffer(size int) *eventBuffer {
	return &eventBuffer{
		bufferSize: size,
		buffer:     make([]*types.Event, 0, size),
		fullChan:   make(chan struct{}),
	}
}

type eventBuffer struct {
	m          sync.Mutex
	bufferSize int
	buffer     []*types.Event

	fullOnce sync.Once
	fullChan chan struct{}
}

func (e *eventBuffer) Get() []*types.Event {
	e.m.Lock()
	defer e.m.Unlock()

	return e.buffer
}

func (e *eventBuffer) Full() <-chan struct{} {
	return e.fullChan
}

func (e *eventBuffer) Append(ev *types.Event) {
	e.m.Lock()
	defer e.m.Unlock()

	if len(e.buffer) >= e.bufferSize {
		e.fullOnce.Do(func() {
			close(e.fullChan)
		})
	} else {
		e.buffer = append(e.buffer, ev)
	}
}
