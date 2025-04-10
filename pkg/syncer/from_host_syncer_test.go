package syncer

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	testResourceName      = "test-name"
	testResourceNamespace = "test-namespace"
)

func TestFromHostSyncer(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	// Initial state of the physical object, before calling any of the sync functions.
	pObject := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testResourceName,
			Namespace:       testResourceNamespace,
			ResourceVersion: syncertesting.FakeClientResourceVersion,
			Labels: map[string]string{
				"example.com/label-a": "test-1",
				"example.com/label-b": "test-2",
			},
			Annotations: map[string]string{
				"example.com/annotation-a": "test-1",
				"example.com/annotation-b": "test-2",
			},
		},
		Data: map[string]string{
			"hello": "test",
		},
	}

	// Initial state of the virtual object, after the initial sync.
	vObject := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testResourceName,
			Namespace:       testResourceNamespace,
			ResourceVersion: syncertesting.FakeClientResourceVersion,
			Labels: map[string]string{
				"example.com/label-a": "test-1",
				"example.com/label-b": "test-2",
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				"example.com/annotation-a":             "test-1",
				"example.com/annotation-b":             "test-2",
				translate.ManagedAnnotationsAnnotation: managedKeysValue(pObject.Annotations),
				translate.ManagedLabelsAnnotation:      managedKeysValue(pObject.Labels),
			},
		},
		Data: map[string]string{
			"hello": "test",
		},
	}

	// Physical object after the update. Here the changes are made only to the metadata, as
	// kind-specific changes (e.g. ConfigMap.Data) are tested in the kind-specific FromHost syncers.
	pObjectUpdated := pObject.DeepCopy()
	pObjectUpdated.Labels["example.com/label-b"] = "updated-test-2"
	pObjectUpdated.Labels["example.com/label-c"] = "new-test-3"
	pObjectUpdated.Annotations["example.com/annotation-a"] = "updated-test-1"
	pObjectUpdated.Annotations["example.com/annotation-c"] = "new-test-3"

	// Virtual object after syncing the updated physical object.
	vObjectUpdated := vObject.DeepCopy()
	vObjectUpdated.Labels["example.com/label-b"] = "updated-test-2"
	vObjectUpdated.Labels["example.com/label-c"] = "new-test-3"
	vObjectUpdated.Annotations["example.com/annotation-a"] = "updated-test-1"
	vObjectUpdated.Annotations["example.com/annotation-c"] = "new-test-3"
	vObjectUpdated.Annotations[translate.ManagedAnnotationsAnnotation] = managedKeysValue(pObjectUpdated.Annotations)
	vObjectUpdated.Annotations[translate.ManagedLabelsAnnotation] = managedKeysValue(pObjectUpdated.Labels)

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Sync new host resource to virtual",
			InitialPhysicalState: []runtime.Object{pObject.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {pObject},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {vObject},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncerCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, NewFakeFromHostSyncer)
				fromHostSyncer := syncer.(*genericFromHostSyncer)
				syncToVirtualEvent := synccontext.NewSyncToVirtualEvent(client.Object(pObject))

				// First call creates the missing namespace.
				result, err := fromHostSyncer.SyncToVirtual(syncerCtx, syncToVirtualEvent)
				assert.Check(t, result.Requeue == true)
				assert.NilError(t, err)

				// Second call creates the virtual resource.
				_, err = fromHostSyncer.SyncToVirtual(syncerCtx, syncToVirtualEvent)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync updated host resource to virtual",
			InitialPhysicalState: []runtime.Object{pObjectUpdated.DeepCopy()},
			InitialVirtualState:  []runtime.Object{vObject.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {pObjectUpdated},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {vObjectUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncerCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, NewFakeFromHostSyncer)
				fromHostSyncer := syncer.(*genericFromHostSyncer)
				syncEvent := synccontext.NewSyncEvent(client.Object(pObjectUpdated.DeepCopy()), client.Object(vObject.DeepCopy()))
				_, err := fromHostSyncer.Sync(syncerCtx, syncEvent)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete virtual resources after host resource has been deleted",
			InitialPhysicalState: []runtime.Object{},                             // host resource has been deleted
			InitialVirtualState:  []runtime.Object{vObject.DeepCopy()},           // virtual resource exists, since it was previously synced
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{}, // virtual resource has been deleted after syncing
			Sync: func(ctx *synccontext.RegisterContext) {
				syncerCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, NewFakeFromHostSyncer)
				fromHostSyncer := syncer.(*genericFromHostSyncer)
				syncToHostEvent := synccontext.NewSyncToHostEvent(client.Object(vObject.DeepCopy()))
				_, err := fromHostSyncer.SyncToHost(syncerCtx, syncToHostEvent)
				assert.NilError(t, err)
			},
		},
	})
}

func managedKeysValue(m map[string]string) string {
	return strings.Join(slices.Sorted(maps.Keys(m)), "\n")
}

// fakeFromHostSyncer is a simple FromHostSyncer test implementation. It mimics
// a ConfigMap FromHost syncer, but it doesn't sync any ConfigMap data from host
// to virtual, since the genericFromHostSyncer tests do not care how kind-specific
// sync is working.
//
// The only important func below is GetMappings, because it specifies how host
// resources are named after being synced to virtual.
type fakeFromHostSyncer struct{}

func NewFakeFromHostSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	gvk, err := apiutil.GVKForObject(&corev1.ConfigMap{}, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}
	fromHostSyncer := &fakeFromHostSyncer{}
	fromHostTranslator, err := translator.NewFromHostTranslatorForGVK(ctx, gvk, fromHostSyncer.GetMappings(ctx.Config.Config))
	if err != nil {
		return nil, err
	}
	return NewFromHost(ctx, fromHostSyncer, fromHostTranslator)
}

func (s *fakeFromHostSyncer) CopyHostObjectToVirtual(_, _ client.Object) {}

func (s *fakeFromHostSyncer) GetProPatches(_ config.Config) []config.TranslatePatch {
	return nil
}

func (s *fakeFromHostSyncer) GetMappings(_ config.Config) map[string]string {
	namePattern := fmt.Sprintf("%s/*", testResourceNamespace)
	return map[string]string{
		namePattern: namePattern,
	}
}
