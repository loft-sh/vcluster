package testing

import (
	"context"
	"fmt"
	"strings"
	"sync"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewScheme creates a new scheme
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	err = volumesnapshotv1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	return scheme
}

// NewFakeClient creates a new fake client for the standard schema
func NewFakeClient(scheme *runtime.Scheme, objs ...runtime.Object) *FakeIndexClient {
	co := []client.Object{}
	for _, o := range objs {
		co = append(co, o.(client.Object))
	}

	return &FakeIndexClient{
		Client:     fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).WithStatusSubresource(co...).Build(),
		scheme:     scheme,
		indexFuncs: map[schema.GroupVersionKind]map[string]client.IndexerFunc{},
		indexes:    map[schema.GroupVersionKind]map[string]map[string][]runtime.Object{},
	}
}

// NewFakeMapper creates a new fake mapper
func NewFakeMapper(scheme *runtime.Scheme) meta.RESTMapper {
	return meta.NewDefaultRESTMapper(scheme.PreferredVersionAllGroups())
}

type FakeIndexClient struct {
	client.Client

	clientLock sync.Mutex
	scheme     *runtime.Scheme

	indexFuncs map[schema.GroupVersionKind]map[string]client.IndexerFunc
	indexes    map[schema.GroupVersionKind]map[string]map[string][]runtime.Object
}

func (fc *FakeIndexClient) updateIndices(ctx context.Context, obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, fc.scheme)
	if err != nil {
		return err
	}

	if _, ok := fc.indexFuncs[gvk]; !ok {
		return nil
	}

	listGvk := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	list, err := fc.scheme.New(listGvk)
	if err != nil {
		return err
	}

	err = fc.Client.List(ctx, list.(client.ObjectList))
	if err != nil {
		return err
	}

	fc.indexes[gvk] = map[string]map[string][]runtime.Object{}
	objs, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	for _, obj := range objs {
		for key, fn := range fc.indexFuncs[gvk] {
			values := fn(obj.(client.Object))
			for _, value := range values {
				if _, ok := fc.indexes[gvk][key]; !ok {
					fc.indexes[gvk][key] = map[string][]runtime.Object{}
				}

				arr, ok := fc.indexes[gvk][key][value]
				if !ok {
					arr = []runtime.Object{}
				}

				arr = append(arr, obj)
				fc.indexes[gvk][key][value] = arr
			}
		}
	}

	return nil
}

func (fc *FakeIndexClient) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	gvk, err := apiutil.GVKForObject(obj, fc.scheme)
	if err != nil {
		return err
	}

	if _, ok := fc.indexFuncs[gvk]; !ok {
		fc.indexFuncs[gvk] = map[string]client.IndexerFunc{}
	}
	fc.indexFuncs[gvk][field] = extractValue
	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	err := fc.Client.Create(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	err := fc.Client.Delete(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	err := fc.Client.Update(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	err := fc.Client.Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}

	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	err := fc.Client.DeleteAllOf(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return fc.updateIndices(ctx, obj)
}

func (fc *FakeIndexClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	return fc.Client.Get(ctx, key, obj, opts...)
}

func (fc *FakeIndexClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	fc.clientLock.Lock()
	defer fc.clientLock.Unlock()

	gvk, err := apiutil.GVKForObject(list, fc.scheme)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(gvk.Kind, "List") {
		return fmt.Errorf("non-list type %T (kind %q) passed as output", list, gvk)
	}

	// we need the non-list GVK, so chop off the "List" from the end of the kind
	gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]

	// Check if we want to list by an index
	for _, opt := range opts {
		matchingFields, ok := opt.(client.MatchingFields)
		if !ok {
			continue
		}

		// Check if we have a value for that
		// TODO: Improve that it works for multiple matching fields
		for k, v := range matchingFields {
			if fc.indexes[gvk] == nil {
				return nil
			}
			if fc.indexes[gvk][k] == nil {
				return nil
			}
			if fc.indexes[gvk][k][v] == nil {
				return nil
			}
			err := meta.SetList(list, fc.indexes[gvk][k][v])
			if err != nil {
				return err
			}

			return nil
		}
	}

	return fc.Client.List(ctx, list, opts...)
}
