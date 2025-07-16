package object

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/dump"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/yaml"
)

// GroupVersionKindNamespacedName uniquely identifies an object
type GroupVersionKindNamespacedName struct {
	schema.GroupVersionKind
	types.NamespacedName
}

// GVKNN returns a GroupVersionKindNamespacedName that uniquely identifies the object
func GVKNN(o client.Object) GroupVersionKindNamespacedName {
	return GroupVersionKindNamespacedName{
		GroupVersionKind: GVK(o),
		NamespacedName:   client.ObjectKeyFromObject(o),
	}
}

func (gvknn GroupVersionKindNamespacedName) String() string {
	str := fmt.Sprintf("%s/%s", gvknn.Group, gvknn.Kind)
	if gvknn.Namespace != "" {
		str += "/" + gvknn.Namespace
	}
	str += "/" + gvknn.Name
	return str
}

func GVK(o runtime.Object) schema.GroupVersionKind {
	return lo.Must(apiutil.GVKForObject(o, scheme.Scheme))
}

func New[T any]() T {
	return reflect.New(reflect.TypeOf(*new(T)).Elem()).Interface().(T)
}

func Unmarshal[T any](raw []byte) *T {
	t := *new(T)
	lo.Must0(yaml.Unmarshal(raw, &t))
	return &t
}

func Hash(o any) string {
	h := fnv.New64a()
	h.Reset()
	fmt.Fprintf(h, "%v", dump.ForHash(o))
	return strconv.FormatUint(h.Sum64(), 10)
}
