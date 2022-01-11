package testing

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FakeClientResourceVersion = "999"
)

func RunTests(t *testing.T, tests []*SyncTest) {
	for _, test := range tests {
		test.Run(t)
	}
}

type Compare func(obj1 runtime.Object, obj2 runtime.Object) bool

type SyncTest struct {
	Name string

	InitialPhysicalState []runtime.Object
	InitialVirtualState  []runtime.Object

	ExpectedPhysicalState map[schema.GroupVersionKind][]runtime.Object
	ExpectedVirtualState  map[schema.GroupVersionKind][]runtime.Object

	Sync    func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger)
	Compare Compare
}

func (s *SyncTest) Run(t *testing.T) {
	scheme := testingutil.NewScheme()
	ctx := context.Background()
	pClient := testingutil.NewFakeClient(scheme, s.InitialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme, s.InitialVirtualState...)

	// do the sync
	s.Sync(ctx, pClient, vClient, scheme, loghelper.New(s.Name))

	// Compare states
	if s.ExpectedPhysicalState != nil {
		for gvk, objs := range s.ExpectedPhysicalState {
			err := compareObjs(ctx, pClient, gvk, scheme, objs, s.Compare)
			if err != nil {
				t.Fatalf("%s - Physical State mismatch: %v", s.Name, err)
			}
		}
	}
	if s.ExpectedVirtualState != nil {
		for gvk, objs := range s.ExpectedVirtualState {
			err := compareObjs(ctx, vClient, gvk, scheme, objs, s.Compare)
			if err != nil {
				t.Fatalf("%s - Virtual State mismatch: %v", s.Name, err)
			}
		}
	}
}

func compareObjs(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, scheme *runtime.Scheme, objs []runtime.Object, compare Compare) error {
	listGvk := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	list, err := scheme.New(listGvk)
	if err != nil {
		return err
	}

	err = c.List(ctx, list.(client.ObjectList))
	if err != nil {
		return err
	}

	existingObjs, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	if len(objs) != len(existingObjs) {
		expectedObjsYaml, err := yaml.Marshal(objs)
		if err != nil {
			return err
		}
		existingObjsYaml, err := yaml.Marshal(existingObjs)
		if err != nil {
			return err
		}

		return fmt.Errorf("expected objs and existing objs length do not match (%d != %d). \n\nExpected: \n%s\n\nExisting: \n%s", len(objs), len(existingObjs), expectedObjsYaml, existingObjsYaml)
	}

	for _, expectedObj := range objs {
		expectedObj = stripObject(expectedObj)
		expectedAccessor, err := meta.Accessor(expectedObj)
		if err != nil {
			return err
		}

		found := false
		for _, existingObjRaw := range existingObjs {
			existingAccessor, err := meta.Accessor(existingObjRaw)
			if err != nil {
				return err
			}

			if expectedAccessor.GetName() == existingAccessor.GetName() && expectedAccessor.GetNamespace() == existingAccessor.GetNamespace() {
				found = true

				// compare objs
				existingObj := stripObject(existingObjRaw)
				isEqual := false
				if compare != nil {
					isEqual = compare(expectedObj, existingObj)
				} else {
					isEqual = apiequality.Semantic.DeepEqual(expectedObj, existingObj)
				}

				if !isEqual {
					expectedObjsYaml, err := yaml.Marshal(expectedObj)
					if err != nil {
						return err
					}
					existingObjsYaml, err := yaml.Marshal(existingObj)
					if err != nil {
						return err
					}

					return fmt.Errorf("expected obj %s/%s and existing obj are different. \n\nExpected: %s\n\nExisting: %s", expectedAccessor.GetNamespace(), expectedAccessor.GetName(), expectedObjsYaml, existingObjsYaml)
				}

				break
			}
		}

		if !found {
			return fmt.Errorf("expected obj %s/%s was not found", expectedAccessor.GetNamespace(), expectedAccessor.GetName())
		}
	}

	return nil
}

func stripObject(obj runtime.Object) runtime.Object {
	newObj := obj.DeepCopyObject()
	accessor, err := meta.Accessor(newObj)
	if err != nil {
		panic(err)
	}

	accessor.SetClusterName("")
	accessor.SetResourceVersion("")
	accessor.SetOwnerReferences(nil)
	accessor.SetGeneration(0)
	accessor.SetUID("")
	accessor.SetSelfLink("")
	accessor.SetManagedFields(nil)

	a := accessor.GetAnnotations()
	if a != nil {
		delete(a, "vcluster.loft.sh/apply")
	}
	if len(a) == 0 {
		accessor.SetAnnotations(nil)
	}

	l := accessor.GetLabels()
	if l != nil {
		delete(l, "vcluster.loft.sh/managed-by")
	}
	if len(l) == 0 {
		accessor.SetLabels(nil)
	}

	typeAccessor, err := meta.TypeAccessor(newObj)
	if err != nil {
		panic(err)
	}

	typeAccessor.SetAPIVersion("")
	typeAccessor.SetKind("")
	return newObj
}
