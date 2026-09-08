package encoding

import (
	"encoding/json"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Decoder is the standard interface for decoding and encoding resources
type Decoder interface {
	Decode(data []byte, defaults *schema.GroupVersionKind) (runtime.Object, error)
	EncodeYAML(obj runtime.Object) ([]byte, error)
	EncodeJSON(obj runtime.Object) ([]byte, error)
}

type decoder struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
}

// NewDecoder creates a new universal decoder
func NewDecoder(scheme *runtime.Scheme, strict bool) Decoder {
	return &decoder{
		scheme: scheme,
		decoder: serializer.NewCodecFactory(scheme, func(options *serializer.CodecFactoryOptions) {
			options.Strict = strict
		}).UniversalDeserializer(),
	}
}

func (d *decoder) Decode(data []byte, defaults *schema.GroupVersionKind) (runtime.Object, error) {
	obj, _, err := d.decoder.Decode(data, defaults, nil)
	if err != nil {
		// Decode into unstructured if the object is not registered
		if runtime.IsNotRegisteredError(err) {
			obj = &unstructured.Unstructured{}
			return obj, yaml.Unmarshal(data, obj)
		}

		return nil, err
	}

	return obj, nil
}

func (d *decoder) EncodeYAML(o runtime.Object) ([]byte, error) {
	return yaml.Marshal(o)
}

func (d *decoder) EncodeJSON(o runtime.Object) ([]byte, error) {
	return json.Marshal(o)
}
