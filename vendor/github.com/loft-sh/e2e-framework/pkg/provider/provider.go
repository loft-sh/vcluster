package provider

import (
	"encoding/json"

	"sigs.k8s.io/e2e-framework/support"
)

type Importable interface {
	support.E2EClusterProvider
	json.Unmarshaler
	json.Marshaler
}

type Type struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func LoadFromBytes(data []byte) (map[string]Importable, error) {
	var raw []json.RawMessage
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Importable)
	for _, rawType := range raw {
		var t Type
		err = json.Unmarshal(rawType, &t)
		if err != nil {
			continue
		}

		if constructor := Get(t.Type); constructor != nil {
			cluster := constructor(t.Name)
			if err := cluster.UnmarshalJSON(rawType); err != nil {
				return nil, err
			}
			result[t.Name] = cluster
		}
	}
	return result, nil
}
