package store

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

func removeMappingFromNameMap(lookupMap *TypedSyncMap[synccontext.Object, lookupName], mapping *Mapping, key synccontext.Object) {
	newLookupName, ok := lookupMap.Load(key)
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
		lookupMap.Delete(key)
		return
	}

	newLookupName.Mappings = newMappings
	lookupMap.Store(key, newLookupName)
}

func addMappingToNameMap(lookupMap *TypedSyncMap[synccontext.Object, lookupName], mapping *Mapping, key, other synccontext.Object) {
	newLookupName, ok := lookupMap.Load(key)
	if !ok {
		newLookupName = lookupName{
			Object: other,
		}
	}

	newLookupName.Mappings = append(newLookupName.Mappings, mapping)
	lookupMap.Store(key, newLookupName)
}
