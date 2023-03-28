package telemetry

import "sync"

func newEventBuffer(size int) *eventBuffer {
	return &eventBuffer{
		bufferSize: size,
		buffer:     make([]*Event, 0, size),
		fullChan:   make(chan struct{}),
	}
}

type eventBuffer struct {
	m          sync.Mutex
	bufferSize int
	buffer     []*Event

	fullOnce sync.Once
	fullChan chan struct{}
}

func (e *eventBuffer) Get() []*Event {
	e.m.Lock()
	defer e.m.Unlock()

	return e.buffer
}

func (e *eventBuffer) Full() <-chan struct{} {
	return e.fullChan
}

func (e *eventBuffer) Append(ev *Event) {
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
