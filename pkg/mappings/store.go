package mappings

import (
	"fmt"
	"sync"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Store struct {
	mappers map[schema.GroupVersionKind]Mapper

	m sync.Mutex
}

func (m *Store) AddMapper(mapper Mapper) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.mappers[mapper.GroupVersionKind()] = mapper
	return nil
}

func (m *Store) ByObject(obj client.Object) Mapper {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		panic(fmt.Sprintf("Couldn't retrieve GVK from object: %v", err))
	}

	return m.ByGVK(gvk)
}

func (m *Store) Has(gvk schema.GroupVersionKind) bool {
	m.m.Lock()
	defer m.m.Unlock()

	_, ok := m.mappers[gvk]
	return ok
}

func (m *Store) ByGVK(gvk schema.GroupVersionKind) Mapper {
	m.m.Lock()
	defer m.m.Unlock()

	mapper, ok := m.mappers[gvk]
	if !ok {
		panic(fmt.Sprintf("Mapper with GVK %s not found", gvk.String()))
	}

	return mapper
}
