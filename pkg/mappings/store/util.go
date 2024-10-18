package store

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

func (s *Store) removeMappingFromNameMap(lookupMap map[synccontext.Object]lookupName, mapping *Mapping, key synccontext.Object) {
	newLookupName, ok := lookupMap[key]
	if !ok {
		return
	}

	// remove from mappings
	newMappings := []*Mapping{}
	for _, otherMapping := range newLookupName.Mappings {
		if otherMapping.String() != mapping.String() {
			newMappings = append(newMappings, otherMapping)
		}
	}
	if len(newMappings) == 0 {
		s.m.Lock()
		delete(lookupMap, key)
		s.m.Unlock()
		return
	}

	newLookupName.Mappings = newMappings
	s.m.Lock()
	lookupMap[key] = newLookupName
	s.m.Unlock()
}

func (s *Store) addMappingToNameMap(lookupMap map[synccontext.Object]lookupName, mapping *Mapping, key, other synccontext.Object) {
	newLookupName, ok := lookupMap[key]
	if !ok {
		newLookupName = lookupName{
			Object: other,
		}
	}

	newLookupName.Mappings = append(newLookupName.Mappings, mapping)
	s.m.Lock()
	lookupMap[key] = newLookupName
	s.m.Unlock()
}
