package clienthelper

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/rest"
	"os"
	"reflect"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
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

	namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}

	return string(namespace), nil
}

// ConvertExtra converts a string array map into the correct kubernetes auth extra value type
func ConvertExtra(orig map[string][]string) map[string]authv1.ExtraValue {
	retMap := map[string]authv1.ExtraValue{}
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
		// TODO: handle runtime.IsNotRegisteredError(err)
		return err
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

func CurrentPodName() (string, error) {
	podNameEnv := os.Getenv("POD_NAME")
	if podNameEnv != "" {
		return podNameEnv, nil
	}

	return os.Hostname()
}

var (
	applyAnnotation = "vcluster.loft.sh/apply"
)

func Apply(ctx context.Context, c client.Client, obj runtime.Object, log loghelper.Logger) error {
	gvk, err := GVKFrom(obj, DefaultScheme)
	if err != nil {
		return err
	}

	oldObj, err := DefaultScheme.New(gvk)
	if err != nil {
		// TODO: handle runtime.IsNotRegisteredError(err)
		return errors.Wrapf(err, "scheme new for %#+v", obj)
	}

	m, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	err = c.Get(ctx, types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      m.GetName(),
	}, oldObj.(client.Object))
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}

		m.SetResourceVersion("")
		editedJS, err := encode(obj)
		if err != nil {
			return err
		}
		newAnnotations := m.GetAnnotations()
		if newAnnotations == nil {
			newAnnotations = map[string]string{}
		}
		newAnnotations[applyAnnotation] = string(editedJS)
		m.SetAnnotations(newAnnotations)
		log.Debugf("create object %s/%s", m.GetNamespace(), m.GetName())
		return c.Create(ctx, obj.(client.Object))
	}

	// make sure typemeta & metadata is aligned
	t, err := meta.TypeAccessor(obj)
	if err != nil {
		return err
	}

	ot, err := meta.TypeAccessor(oldObj)
	if err != nil {
		return err
	}

	t.SetAPIVersion(ot.GetAPIVersion())
	t.SetKind(ot.GetKind())

	om, err := meta.Accessor(oldObj)
	if err != nil {
		return err
	}

	// make sure resource versions match
	m.SetResourceVersion(om.GetResourceVersion())

	annotations := om.GetAnnotations()
	var originalJS []byte
	if annotations != nil && annotations[applyAnnotation] != "" {
		originalJS = []byte(annotations[applyAnnotation])
	}

	// create patch if changed
	currentJS, err := encode(oldObj)
	if err != nil {
		return err
	}

	editedJS, err := encode(obj)
	if err != nil {
		return err
	}

	newAnnotations := m.GetAnnotations()
	if newAnnotations == nil {
		newAnnotations = map[string]string{}
	}
	newAnnotations[applyAnnotation] = string(editedJS)
	m.SetAnnotations(newAnnotations)

	editedWithAnnotationJS, err := encode(obj)
	if err != nil {
		return err
	}

	patchType := types.StrategicMergePatchType
	var patch []byte
	if originalJS != nil {
		if reflect.DeepEqual(originalJS, editedJS) {
			// no edit, so just skip it.
			return nil
		}

		lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(obj)
		if err != nil {
			return err
		}
		patch, err = strategicpatch.CreateThreeWayMergePatch(originalJS, editedWithAnnotationJS, currentJS, lookupPatchMeta, true)
		if err != nil {
			return err
		}
	} else {
		if reflect.DeepEqual(currentJS, editedJS) {
			// no edit, so just skip it.
			return nil
		}

		patch, err = strategicpatch.CreateTwoWayMergePatch(currentJS, editedWithAnnotationJS, obj)
		if err != nil {
			return err
		}
	}

	log.Debugf("update object %s/%s", m.GetNamespace(), m.GetName())
	return c.Patch(ctx, oldObj.(client.Object), client.RawPatch(patchType, patch))
}

func encode(obj runtime.Object) ([]byte, error) {
	serialization, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		return nil, err
	}
	js, err := yaml.ToJSON(serialization)
	if err != nil {
		return nil, err
	}
	return js, nil
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
