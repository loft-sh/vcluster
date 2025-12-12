package provider

import (
	"encoding/json"

	"github.com/loft-sh/e2e-framework/pkg/provider/kind"
	"github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"sigs.k8s.io/e2e-framework/support"
)

type Importable interface {
	support.E2EClusterProvider
	json.Unmarshaler
	json.Marshaler
}

var _ Importable = &kind.Cluster{}
var _ Importable = &vcluster.Cluster{}

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

		switch t.Type {
		case vcluster.Type:
			cluster := vcluster.NewCluster(t.Name)
			if err := cluster.UnmarshalJSON(rawType); err != nil {
				return nil, err
			}
			result[t.Name] = cluster
		case kind.Type:
			cluster := kind.NewCluster(t.Name)
			if err := cluster.UnmarshalJSON(rawType); err != nil {
				return nil, err
			}
			result[t.Name] = cluster
		default:
			continue
		}
	}
	return result, nil
}
