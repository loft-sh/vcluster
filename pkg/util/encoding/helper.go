package encoding

import (
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Convert converts the from object into the to object
func Convert(from runtime.Object, to runtime.Object) error {
	out, err := yaml.Marshal(from)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(out, to)
	if err != nil {
		return err
	}

	// remove api version and kind
	t, err := meta.TypeAccessor(to)
	if err != nil {
		return err
	}

	t.SetAPIVersion("")
	t.SetKind("")
	return nil
}

// ConvertList converts the objects from the from list and puts them into the to list
func ConvertList(fromList runtime.Object, toList runtime.Object, new rest.Storage) error {
	list, err := meta.ExtractList(fromList)
	if err != nil {
		return err
	}

	newItems := []runtime.Object{}
	for _, item := range list {
		newItem := new.New()
		err = Convert(item, newItem)
		if err != nil {
			return err
		}

		newItems = append(newItems, newItem)
	}

	return meta.SetList(toList, newItems)
}
