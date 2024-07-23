package store

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

func addMappingToLabelMap(lookupMap map[string]lookupLabel, mapping *Mapping, key, other string) {
	newLookupLabel, ok := lookupMap[key]
	if !ok {
		newLookupLabel = lookupLabel{
			Label: other,
		}
	}

	newLookupLabel.Mappings = append(newLookupLabel.Mappings, mapping)
	lookupMap[key] = newLookupLabel
}

func removeMappingFromNameMap(lookupMap map[synccontext.Object]lookupName, mapping *Mapping, key synccontext.Object) {
	newLookupName, ok := lookupMap[key]
	if !ok {
		return
	}

	// remove from mappings
	newMappings := []*Mapping{}
	for _, otherMapping := range newLookupName.Mappings {
		if otherMapping != mapping {
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

func removeMappingFromLabelMap(lookupMap map[string]lookupLabel, mapping *Mapping, key string) {
	newLookupLabel, ok := lookupMap[key]
	if !ok {
		return
	}

	// remove from mappings
	newMappings := []*Mapping{}
	for _, otherMapping := range newLookupLabel.Mappings {
		if otherMapping != mapping {
			newMappings = append(newMappings, otherMapping)
		}
	}
	if len(newMappings) == 0 {
		delete(lookupMap, key)
		return
	}

	newLookupLabel.Mappings = newMappings
	lookupMap[key] = newLookupLabel
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
