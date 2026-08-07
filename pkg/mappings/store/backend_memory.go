package store

import (
	"context"
	"sync"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

func NewMemoryBackend(mappings ...*Mapping) Backend {
	internalMap := map[synccontext.NameMapping]*Mapping{}
	for _, m := range mappings {
		internalMap[m.NameMapping] = m
	}

	return &memoryBackend{
		mappings: internalMap,
	}
}

type memoryBackend struct {
	m sync.Mutex

	mappings map[synccontext.NameMapping]*Mapping

	watches []chan BackendWatchResponse
}

func (m *memoryBackend) List(_ context.Context) ([]*Mapping, error) {
	m.m.Lock()
	defer m.m.Unlock()

	retMappings := make([]*Mapping, 0, len(m.mappings))
	for _, mapping := range m.mappings {
		retMappings = append(retMappings, mapping)
	}

	return retMappings, nil
}

func (m *memoryBackend) Watch(ctx context.Context) <-chan BackendWatchResponse {
	m.m.Lock()
	defer m.m.Unlock()

	watchChan := make(chan BackendWatchResponse)
	m.watches = append(m.watches, watchChan)
	go func() {
		<-ctx.Done()

		m.m.Lock()
		defer m.m.Unlock()

		// remove chan
		close(watchChan)

		// remove from slice
		newWatches := make([]chan BackendWatchResponse, 0, len(m.watches)-1)
		for _, watch := range m.watches {
			if watch != watchChan {
				newWatches = append(newWatches, watch)
			}
		}
		m.watches = newWatches
	}()

	return watchChan
}

func (m *memoryBackend) Save(_ context.Context, mapping *Mapping) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.mappings[mapping.NameMapping] = mapping
	for _, watchChan := range m.watches {
		go func(watchChan chan BackendWatchResponse) {
			watchChan <- BackendWatchResponse{
				Events: []*BackendWatchEvent{
					{
						Type:    BackendWatchEventTypeUpdate,
						Mapping: mapping,
					},
				},
			}
		}(watchChan)
	}

	return nil
}

func (m *memoryBackend) Delete(_ context.Context, mapping *Mapping) error {
	m.m.Lock()
	defer m.m.Unlock()

	delete(m.mappings, mapping.NameMapping)
	for _, watchChan := range m.watches {
		go func(watchChan chan BackendWatchResponse) {
			watchChan <- BackendWatchResponse{
				Events: []*BackendWatchEvent{
					{
						Type:    BackendWatchEventTypeDelete,
						Mapping: mapping,
					},
				},
			}
		}(watchChan)
	}

	return nil
}

func (m *memoryBackend) DeleteReconstructed(_ context.Context, mapping *Mapping) error {
	m.m.Lock()
	defer m.m.Unlock()

	delete(m.mappings, mapping.NameMapping)
	for _, watchChan := range m.watches {
		go func(watchChan chan BackendWatchResponse) {
			watchChan <- BackendWatchResponse{
				Events: []*BackendWatchEvent{
					{
						Type: BackendWatchEventTypeDeleteReconstructed,
						Mapping: &Mapping{
							NameMapping: synccontext.NameMapping{
								GroupVersionKind: mapping.GroupVersionKind,
								VirtualName:      mapping.VirtualName,
							},
						},
					},
				},
			}
		}(watchChan)
	}

	return nil
}
