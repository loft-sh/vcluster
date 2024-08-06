package store

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

func removeMappingFromNameMap(lookupMap map[synccontext.Object]lookupName, mapping *Mapping, key synccontext.Object) {
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
		delete(lookupMap, key)
		return
	}

	newLookupName.Mappings = newMappings
	lookupMap[key] = newLookupName
}

func addMappingToNameMap(lookupMap map[synccontext.Object]lookupName, mapping *Mapping, key, other synccontext.Object) {
	newLookupName, ok := lookupMap[key]
	if !ok {
		newLookupName = lookupName{
			Object: other,
		}
	}

	newLookupName.Mappings = append(newLookupName.Mappings, mapping)
	lookupMap[key] = newLookupName
}
