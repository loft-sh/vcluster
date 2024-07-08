package clienthelper

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultScheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(DefaultScheme)
	// API extensions are not in the above scheme set,
	// and must thus be added separately.
	_ = authenticationv1.AddToScheme(DefaultScheme)
	_ = apiextensionsv1.AddToScheme(DefaultScheme)
	_ = apiregistrationv1.AddToScheme(DefaultScheme)
}

func CurrentNamespace() (string, error) {
	namespaceEnv := os.Getenv("NAMESPACE")
	if namespaceEnv != "" {
		return namespaceEnv, nil
	}

	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}

	return string(namespace), nil
}

// ConvertExtra converts a string array map into the correct kubernetes auth extra value type
func ConvertExtra(orig map[string][]string) map[string]authorizationv1.ExtraValue {
	retMap := map[string]authorizationv1.ExtraValue{}
	for k, v := range orig {
		retMap[k] = v
	}
	return retMap
}

// ConvertExtraFrom converts a string array map into the correct kubernetes auth extra value type
func ConvertExtraFrom(orig map[string]authenticationv1.ExtraValue) map[string][]string {
	retMap := map[string][]string{}
	for k, v := range orig {
		retMap[k] = v
	}
	return retMap
}

func GetByIndex(ctx context.Context, c client.Client, obj runtime.Object, index, value string) error {
	gvk, err := GVKFrom(obj, c.Scheme())
	if err != nil {
		return err
	}

	list, err := c.Scheme().New(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	if err != nil {
		if !runtime.IsNotRegisteredError(err) {
			return err
		}

		unstructuredList := &unstructured.UnstructuredList{}
		unstructuredList.SetKind(gvk.Kind + "List")
		unstructuredList.SetAPIVersion(gvk.GroupVersion().String())
		list = unstructuredList
	}

	err = c.List(ctx, list.(client.ObjectList), client.MatchingFields{index: value})
	if err != nil {
		return err
	}

	objs, err := meta.ExtractList(list)
	if err != nil {
		return err
	} else if len(objs) == 0 {
		return kerrors.NewNotFound(schema.GroupResource{Group: gvk.Group}, value)
	} else if len(objs) > 1 {
		return kerrors.NewConflict(schema.GroupResource{Group: gvk.Group}, value, fmt.Errorf("more than 1 object with the value"))
	}

	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("object not a pointer")
	}

	val = val.Elem()
	newVal := reflect.Indirect(reflect.ValueOf(objs[0]))
	if !val.Type().AssignableTo(newVal.Type()) {
		return fmt.Errorf("mismatched types")
	}

	val.Set(newVal)
	return nil
}

func GVKFrom(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	} else if len(gvks) != 1 {
		return schema.GroupVersionKind{}, fmt.Errorf("unexpected number of object kinds: %d", len(gvks))
	}

	return gvks[0], nil
}

func NewImpersonatingClient(config *rest.Config, mapper meta.RESTMapper, user user.Info, scheme *runtime.Scheme) (client.Client, error) {
	// Impersonate user
	restConfig := rest.CopyConfig(config)
	restConfig.Impersonate.UserName = user.GetName()
	restConfig.Impersonate.Groups = user.GetGroups()
	restConfig.Impersonate.Extra = user.GetExtra()

	// Create client
	return client.New(restConfig, client.Options{Scheme: scheme, Mapper: mapper})
}
