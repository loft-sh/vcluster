package testing

import (
	"context"
	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"testing"
)

func FakeStartSyncer(t *testing.T, ctx *synccontext.RegisterContext, create func(ctx *synccontext.RegisterContext) (syncer.Object, error)) (*synccontext.SyncContext, syncer.Object) {
	object, err := create(ctx)
	assert.NilError(t, err)

	// run register indices
	registerer, ok := object.(syncer.IndicesRegisterer)
	if ok {
		err := registerer.RegisterIndices(ctx)
		assert.NilError(t, err)
	}

	return synccontext.ConvertContext(ctx, object.Name()), object
}

func NewFakeRegisterContext(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	return &synccontext.RegisterContext{
		Context: context.TODO(),
		Options: &controllercontext.VirtualClusterOptions{
			Name:            "vcluster",
			ServiceName:     "vcluster",
			TargetNamespace: "test",
		},
		Controllers:            controllercontext.ExistingControllers,
		NodeServiceProvider:    &fakeNodeServiceProvider{},
		LockFactory:            locks.NewDefaultLockFactory(),
		TargetNamespace:        "test",
		CurrentNamespace:       "test",
		CurrentNamespaceClient: pClient,
		VirtualManager:         newFakeManager(vClient),
		PhysicalManager:        newFakeManager(pClient),
	}
}

type fakeNodeServiceProvider struct{}

func (f *fakeNodeServiceProvider) Start(ctx context.Context) {}
func (f *fakeNodeServiceProvider) Lock()                     {}
func (f *fakeNodeServiceProvider) Unlock()                   {}
func (f *fakeNodeServiceProvider) GetNodeIP(ctx context.Context, name types.NamespacedName) (string, error) {
	return "127.0.0.1", nil
}

func newFakeEventBroadcaster() record.EventBroadcaster {
	return &fakeEventBroadcaster{}
}

type fakeEventBroadcaster struct{}

func (f *fakeEventBroadcaster) StartEventWatcher(eventHandler func(*corev1.Event)) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartRecordingToSink(sink record.EventSink) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartLogging(logf func(format string, args ...interface{})) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartStructuredLogging(verbosity klog.Level) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) NewRecorder(scheme *runtime.Scheme, source corev1.EventSource) record.EventRecorder {
	return f
}

func (f *fakeEventBroadcaster) Shutdown() {}

func (f *fakeEventBroadcaster) Event(object runtime.Object, eventtype, reason, message string) {}

func (f *fakeEventBroadcaster) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}

func (f *fakeEventBroadcaster) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
