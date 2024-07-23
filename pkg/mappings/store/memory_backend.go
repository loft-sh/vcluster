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
}

func (m *memoryBackend) List(_ context.Context) ([]*Mapping, error) {
	m.m.Lock()
	defer m.m.Unlock()

	retMappings := []*Mapping{}
	for _, mapping := range m.mappings {
		retMappings = append(retMappings, mapping)
	}

	return retMappings, nil
}

func (m *memoryBackend) Watch(_ context.Context) <-chan BackendWatchResponse {
	m.m.Lock()
	defer m.m.Unlock()

	return make(chan BackendWatchResponse)
}

func (m *memoryBackend) Save(_ context.Context, mapping *Mapping) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.mappings[mapping.NameMapping] = mapping
	return nil
}

func (m *memoryBackend) Delete(_ context.Context, mapping *Mapping) error {
	m.m.Lock()
	defer m.m.Unlock()

	delete(m.mappings, mapping.NameMapping)
	return nil
}
