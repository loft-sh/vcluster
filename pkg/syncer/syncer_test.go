package syncer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// named mock instead of fake because there's a real "fake" syncer that syncs fake objects
type mockSyncer struct {
	syncertypes.GenericTranslator
}

func NewMockSyncer(ctx *synccontext.RegisterContext) (syncertypes.Syncer, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Secrets())
	if err != nil {
		return nil, err
	}

	return &mockSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "secrets", &corev1.Secret{}, mapper),
	}, nil
}

func (s *mockSyncer) Syncer() syncertypes.Sync[client.Object] {
	return ToGenericSyncer[*corev1.Secret](s)
}

// SyncToHost is called when a virtual object was created and needs to be synced down to the physical cluster
func (s *mockSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Secret]) (ctrl.Result, error) {
	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.GetName(), Namespace: event.Virtual.GetNamespace()}, event.Virtual))
	if pObj == nil {
		return ctrl.Result{}, errors.New("naive translate create failed")
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), false)
}

// Sync is called to sync a virtual object with a physical object
func (s *mockSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)
	event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)

	// check data
	event.Virtual.Data, event.Host.Data = patcher.CopyBidirectional(
		event.VirtualOld.Data,
		event.Virtual.Data,
		event.HostOld.Data,
		event.Host.Data,
	)

	return ctrl.Result{}, nil
}

func (s *mockSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}

var _ syncertypes.Syncer = &mockSyncer{}

var (
	namespaceInVClusterA = "default"
)

type fakeSource struct {
	m sync.Mutex

	queue workqueue.TypedRateLimitingInterface[ctrl.Request]
}

func (f *fakeSource) String() string {
	return "fake-source"
}

func (f *fakeSource) Add(request reconcile.Request) {
	f.m.Lock()
	defer f.m.Unlock()

	f.queue.Add(request)
}

func (f *fakeSource) Start(_ context.Context, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) error {
	f.m.Lock()
	defer f.m.Unlock()

	f.queue = queue
	return nil
}

func TestController(t *testing.T) {
	translator := translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	type testCase struct {
		Name string

		EnqueueObjs []types.NamespacedName

		InitialPhysicalState []runtime.Object
		InitialVirtualState  []runtime.Object

		ExpectedPhysicalState map[schema.GroupVersionKind][]runtime.Object
		ExpectedVirtualState  map[schema.GroupVersionKind][]runtime.Object

		Compare syncertesting.Compare
	}

	testCases := []testCase{
		{
			Name: "should sync down",

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "123",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: testingutil.DefaultTestTargetNamespace,
						UID:       "123",
					},
				},
			},

			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: namespaceInVClusterA,
							UID:       "123",
						},
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: testingutil.DefaultTestTargetNamespace,
							UID:       "123",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
							Namespace: testingutil.DefaultTestTargetNamespace,
							Annotations: map[string]string{
								translate.NameAnnotation:          "a",
								translate.NamespaceAnnotation:     namespaceInVClusterA,
								translate.UIDAnnotation:           "123",
								translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
								translate.HostNameAnnotation:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
								translate.HostNamespaceAnnotation: testingutil.DefaultTestTargetNamespace,
							},
							Labels: map[string]string{
								translate.NamespaceLabel: namespaceInVClusterA,
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Logf("running test #%d: %s", i, tc.Name)

		// setup mocks
		ctx := context.Background()
		pClient := testingutil.NewFakeClient(scheme.Scheme, tc.InitialPhysicalState...)
		vClient := testingutil.NewFakeClient(scheme.Scheme, tc.InitialVirtualState...)

		fakeContext := syncertesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
		syncer, err := NewMockSyncer(fakeContext)
		assert.NilError(t, err)

		assert.Check(t, fakeContext.Config.HostConfig != nil)

		syncController, err := NewSyncController(fakeContext, syncer)
		assert.NilError(t, err)

		genericController, err := syncController.Build(fakeContext)
		assert.NilError(t, err)

		source := &fakeSource{}
		err = genericController.Watch(source)
		assert.NilError(t, err)

		go func() {
			err = genericController.Start(fakeContext)
			assert.NilError(t, err)
		}()

		time.Sleep(time.Second)

		// execute
		for _, req := range tc.EnqueueObjs {
			source.Add(ctrl.Request{NamespacedName: req})
		}

		time.Sleep(time.Second)

		// assert expected result
		// Compare states
		if tc.ExpectedPhysicalState != nil {
			for gvk, objs := range tc.ExpectedPhysicalState {
				err := syncertesting.CompareObjs(ctx, t, tc.Name+" physical state", fakeContext.HostManager.GetClient(), gvk, scheme.Scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Physical State mismatch: %v", tc.Name, err)
				}
			}
		}
		if tc.ExpectedVirtualState != nil {
			for gvk, objs := range tc.ExpectedVirtualState {
				err := syncertesting.CompareObjs(ctx, t, tc.Name+" virtual state", fakeContext.VirtualManager.GetClient(), gvk, scheme.Scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Virtual State mismatch: %v", tc.Name, err)
				}
			}
		}
	}
}

func TestReconcile(t *testing.T) {
	translator := translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	type testCase struct {
		Name  string
		Focus bool

		Syncer func(ctx *synccontext.RegisterContext) (syncertypes.Syncer, error)

		EnqueueObjs []types.NamespacedName

		InitialPhysicalState []runtime.Object
		InitialVirtualState  []runtime.Object

		CreatePhysicalObjects []client.Object
		CreateVirtualObjects  []client.Object

		ExpectedPhysicalState map[schema.GroupVersionKind][]runtime.Object
		ExpectedVirtualState  map[schema.GroupVersionKind][]runtime.Object

		Compare syncertesting.Compare

		// ExpectedEvents are the events recorded on virtual objects, nil skips the check
		ExpectedEvents []string

		shouldErr bool
		errMsg    string
	}

	testCases := []testCase{
		{
			Name:   "should sync down",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "123",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: testingutil.DefaultTestTargetNamespace,
						UID:       "123",
					},
				},
			},

			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: namespaceInVClusterA,
							UID:       "123",
						},
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: testingutil.DefaultTestTargetNamespace,
							UID:       "123",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
							Namespace: testingutil.DefaultTestTargetNamespace,
							Annotations: map[string]string{
								translate.NameAnnotation:          "a",
								translate.NamespaceAnnotation:     namespaceInVClusterA,
								translate.UIDAnnotation:           "123",
								translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
								translate.HostNamespaceAnnotation: testingutil.DefaultTestTargetNamespace,
								translate.HostNameAnnotation:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
							},
							Labels: map[string]string{
								translate.NamespaceLabel: namespaceInVClusterA,
							},
						},
					},
				},
			},

			shouldErr: false,
		},
		{
			Name:   "should adopt object of desired name when already exists",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "123",
					},
					Data: map[string][]byte{
						"datakey1": []byte("datavalue1"),
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// existing object doesn't have annotations/labels indicating it is owned, but has the name of the synced object
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
						Namespace: testingutil.DefaultTestTargetNamespace,
						Annotations: map[string]string{
							"app": "existing",
						},
						Labels: map[string]string{
							"app": "existing",
						},
					},
					Data: map[string][]byte{
						"datakey1": []byte("datavalue1"),
					},
				},
			},

			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a",
							Namespace: namespaceInVClusterA,
							UID:       "123",
						},
						Data: map[string][]byte{
							"datakey1": []byte("datavalue1"),
						},
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
							Namespace: testingutil.DefaultTestTargetNamespace,
							Annotations: map[string]string{
								"app":                             "existing",
								translate.NameAnnotation:          "a",
								translate.NamespaceAnnotation:     namespaceInVClusterA,
								translate.UIDAnnotation:           "123",
								translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
								translate.HostNameAnnotation:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
								translate.HostNamespaceAnnotation: testingutil.DefaultTestTargetNamespace,
							},
							Labels: map[string]string{
								translate.NamespaceLabel: namespaceInVClusterA,
							},
						},
						Data: map[string][]byte{
							"datakey1": []byte("datavalue1"),
						},
					},
				},
			},
		},
		{
			Name:   "should not adopt virtual object",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "abc", Namespace: testingutil.DefaultTestTargetNamespace},
			},

			CreateVirtualObjects: []client.Object{
				// secret that might be created by ingress controller or cert managers
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "abc",
						Namespace: namespaceInVClusterA,
						UID:       "123",
					},
				},
			},

			CreatePhysicalObjects: []client.Object{
				// existing object doesn't have annotations/labels indicating it is owned, but has the name of the synced object
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "abc",
						Namespace: testingutil.DefaultTestTargetNamespace,
						Annotations: map[string]string{
							translate.NameAnnotation:      "abc",
							translate.NamespaceAnnotation: namespaceInVClusterA,
						},
						Labels: map[string]string{
							translate.MarkerLabel: translate.VClusterName,
						},
					},
					Data: map[string][]byte{
						"datakey1": []byte("datavalue1"),
					},
				},
			},

			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "abc",
							Namespace: namespaceInVClusterA,
							UID:       "123",
						},
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// existing secret should remain
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "abc",
							Namespace: testingutil.DefaultTestTargetNamespace,
							Annotations: map[string]string{
								translate.NameAnnotation:      "abc",
								translate.NamespaceAnnotation: namespaceInVClusterA,
							},
							Labels: map[string]string{
								translate.MarkerLabel: translate.VClusterName,
							},
						},
						Data: map[string][]byte{
							"datakey1": []byte("datavalue1"),
						},
					},
				},
			},
		},
		{
			Name:   "should not touch host object that belongs to another virtual object",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "456",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// host object with the same host name, synced from a different virtual object
				otherVirtualObjectHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// host object should remain untouched
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					otherVirtualObjectHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name),
				},
			},

			// the event must not reveal the host name
			ExpectedEvents: []string{
				"Warning SyncError default/a: Another object in this cluster already maps to the same host name, rename this object to sync it",
			},
		},
		{
			Name:   "should not delete host object of another virtual object when virtual object is missing",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialPhysicalState: []runtime.Object{
				// host object with the same host name, synced from a different virtual object
				otherVirtualObjectHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// host object should remain untouched
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					otherVirtualObjectHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name),
				},
			},

			// there is no virtual object to record an event on
			ExpectedEvents: []string{},
		},
		{
			Name:   "should sync host object of another kind with the same name",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "456",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// host object synced from a virtual object of another kind is left to the existing sync logic
				otherVirtualObjectHostSecretWithKind(translator.HostName(nil, "a", namespaceInVClusterA).Name, corev1.SchemeGroupVersion.WithKind("ConfigMap").String()),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// host object is synced from the requested virtual object
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
							Namespace: testingutil.DefaultTestTargetNamespace,
							Annotations: map[string]string{
								translate.NameAnnotation:          "a",
								translate.NamespaceAnnotation:     namespaceInVClusterA,
								translate.UIDAnnotation:           "456",
								translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
								translate.HostNameAnnotation:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
								translate.HostNamespaceAnnotation: testingutil.DefaultTestTargetNamespace,
							},
							Labels: map[string]string{
								translate.NamespaceLabel: namespaceInVClusterA,
							},
						},
						Data: map[string][]byte{
							"datakey1": []byte("datavalue1"),
						},
					},
				},
			},

			ExpectedEvents: []string{},
		},
		{
			Name:   "should delete stale host object of the same virtual object",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				// virtual object was recreated, so its uid changed
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "456",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				ownHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name, "123", "datavalue1"),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// stale host object should be deleted
				corev1.SchemeGroupVersion.WithKind("Secret"): {},
			},

			ExpectedEvents: []string{},
		},
		{
			Name:   "should sync host object of the same virtual object",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "123",
					},
					Data: map[string][]byte{
						"datakey1": []byte("datavalue1"),
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// host object synced from the requested virtual object, without the host name annotations and labels
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      translator.HostName(nil, "a", namespaceInVClusterA).Name,
						Namespace: testingutil.DefaultTestTargetNamespace,
						Annotations: map[string]string{
							translate.NameAnnotation:      "a",
							translate.NamespaceAnnotation: namespaceInVClusterA,
							translate.UIDAnnotation:       "123",
							translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Secret").String(),
						},
					},
					Data: map[string][]byte{
						"datakey1": []byte("datavalue1"),
					},
				},
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// sync should add the missing annotations and labels
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					ownHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name, "123", "datavalue1"),
				},
			},

			ExpectedEvents: []string{},
		},
		{
			Name:   "should not touch host object of another virtual object without kind annotation",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialVirtualState: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a",
						Namespace: namespaceInVClusterA,
						UID:       "456",
					},
				},
			},

			InitialPhysicalState: []runtime.Object{
				// host objects synced by older versions might not have a kind annotation
				otherVirtualObjectHostSecretWithKind(translator.HostName(nil, "a", namespaceInVClusterA).Name, ""),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// host object should remain untouched
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					otherVirtualObjectHostSecretWithKind(translator.HostName(nil, "a", namespaceInVClusterA).Name, ""),
				},
			},

			ExpectedEvents: []string{
				"Warning SyncError default/a: Another object in this cluster already maps to the same host name, rename this object to sync it",
			},
		},
		{
			Name:   "should delete host object of the same virtual object when virtual object is missing",
			Syncer: NewMockSyncer,

			EnqueueObjs: []types.NamespacedName{
				{Name: "a", Namespace: namespaceInVClusterA},
			},

			InitialPhysicalState: []runtime.Object{
				// without the virtual object there is no name mapping, so the host object needs the marker label to be managed
				func() *corev1.Secret {
					secret := ownHostSecret(translator.HostName(nil, "a", namespaceInVClusterA).Name, "123", "datavalue1")
					secret.Labels[translate.MarkerLabel] = translate.VClusterName
					return secret
				}(),
			},

			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				// host object of the deleted virtual object should be deleted
				corev1.SchemeGroupVersion.WithKind("Secret"): {},
			},

			ExpectedEvents: []string{},
		},
	}
	sort.SliceStable(testCases, func(i, j int) bool {
		// place focused tests first
		return testCases[i].Focus && !testCases[j].Focus
	})
	// record if any tests were focused
	hasFocus := false
	for i, tc := range testCases {
		t.Logf("running test #%d: %s", i, tc.Name)
		if tc.Focus {
			hasFocus = true
			t.Log("test is focused")
		} else if hasFocus { // fail if any tests were focused
			t.Fatal("some tests are focused")
		}

		// testing scenario:
		// virt object queued (existing, nonexisting)
		// corresponding physical object (nil, not-nil)

		// setup mocks
		options := &syncertypes.Options{}
		ctx := context.Background()
		pClient := testingutil.NewFakeClient(scheme.Scheme, tc.InitialPhysicalState...)
		vClient := testingutil.NewFakeClient(scheme.Scheme, tc.InitialVirtualState...)

		fakeContext := syncertesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
		syncer, err := tc.Syncer(fakeContext)
		assert.NilError(t, err)

		eventRecorder := &recordingEventRecorder{events: []string{}}
		controller := &SyncController{
			syncer: syncer,

			genericSyncer: syncer.Syncer(),

			log:            loghelper.New(syncer.Name()),
			vEventRecorder: eventRecorder,
			physicalClient: pClient,

			hostNameRequestLookup: map[ctrl.Request]ctrl.Request{},

			currentNamespace:       fakeContext.CurrentNamespace,
			currentNamespaceClient: fakeContext.CurrentNamespaceClient,

			mappings: fakeContext.Mappings,

			virtualClient: vClient,
			options:       options,

			objectCache: synccontext.NewBidirectionalObjectCache(syncer.Resource(), syncer),
			config:      fakeContext.Config,
		}

		// create objects
		for _, pObj := range tc.CreatePhysicalObjects {
			err = fakeContext.HostManager.GetClient().Create(ctx, pObj)
			assert.NilError(t, err)
		}
		for _, vObj := range tc.CreateVirtualObjects {
			err = fakeContext.VirtualManager.GetClient().Create(ctx, vObj)
			assert.NilError(t, err)
		}

		// execute
		for _, req := range tc.EnqueueObjs {
			_, err = controller.Reconcile(ctx, ctrl.Request{NamespacedName: req})
		}
		if tc.shouldErr {
			assert.ErrorContains(t, err, tc.errMsg)
		} else {
			assert.NilError(t, err)
		}

		// assert expected result
		// Compare states
		if tc.ExpectedPhysicalState != nil {
			for gvk, objs := range tc.ExpectedPhysicalState {
				err := syncertesting.CompareObjs(ctx, t, tc.Name+" physical state", fakeContext.HostManager.GetClient(), gvk, scheme.Scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Physical State mismatch: %v", tc.Name, err)
				}
			}
		}
		if tc.ExpectedVirtualState != nil {
			for gvk, objs := range tc.ExpectedVirtualState {
				err := syncertesting.CompareObjs(ctx, t, tc.Name+" virtual state", fakeContext.VirtualManager.GetClient(), gvk, scheme.Scheme, objs, tc.Compare)
				if err != nil {
					t.Fatalf("%s - Virtual State mismatch: %v", tc.Name, err)
				}
			}
		}
		if tc.ExpectedEvents != nil {
			assert.DeepEqual(t, eventRecorder.events, tc.ExpectedEvents)
		}
	}
}

// recordingEventRecorder records events as "<type> <reason> <namespace>/<name>: <note>"
type recordingEventRecorder struct {
	events []string
}

func (r *recordingEventRecorder) Eventf(regarding runtime.Object, _ runtime.Object, eventtype, reason, _, note string, args ...interface{}) {
	obj := regarding.(client.Object)
	r.events = append(r.events, fmt.Sprintf("%s %s %s/%s: %s", eventtype, reason, obj.GetNamespace(), obj.GetName(), fmt.Sprintf(note, args...)))
}

func ownHostSecret(name, uid, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testingutil.DefaultTestTargetNamespace,
			Annotations: map[string]string{
				translate.NameAnnotation:          "a",
				translate.NamespaceAnnotation:     namespaceInVClusterA,
				translate.UIDAnnotation:           uid,
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
				translate.HostNameAnnotation:      name,
				translate.HostNamespaceAnnotation: testingutil.DefaultTestTargetNamespace,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: namespaceInVClusterA,
			},
		},
		Data: map[string][]byte{
			"datakey1": []byte(data),
		},
	}
}

func otherVirtualObjectHostSecret(name string) *corev1.Secret {
	return otherVirtualObjectHostSecretWithKind(name, corev1.SchemeGroupVersion.WithKind("Secret").String())
}

// otherVirtualObjectHostSecretWithKind returns a host secret synced from other-namespace/other, without a kind annotation if kind is empty
func otherVirtualObjectHostSecretWithKind(name, kind string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testingutil.DefaultTestTargetNamespace,
			Annotations: map[string]string{
				translate.NameAnnotation:      "other",
				translate.NamespaceAnnotation: "other-namespace",
				translate.UIDAnnotation:       "123",
				translate.KindAnnotation:      kind,
			},
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
		},
		Data: map[string][]byte{
			"datakey1": []byte("datavalue1"),
		},
	}
	if kind == "" {
		delete(secret.Annotations, translate.KindAnnotation)
	}
	return secret
}
