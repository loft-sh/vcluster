package testing

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"testing"

	"github.com/loft-sh/vcluster/pkg/util/log"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"

	controllercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

const (
	DefaultTestTargetNamespace     = "test"
	DefaultTestCurrentNamespace    = "vcluster"
	DefaultTestVclusterName        = "vcluster"
	DefaultTestVclusterServiceName = "vcluster"
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

	syncCtx := synccontext.ConvertContext(ctx, object.Name())
	syncCtx.Log = loghelper.NewFromExisting(log.NewLog(0), object.Name())
	return syncCtx, object
}

func NewFakeRegisterContext(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	translate.Default = translate.NewSingleNamespaceTranslator(DefaultTestTargetNamespace)
	return &synccontext.RegisterContext{
		Context: context.TODO(),
		Options: &controllercontext.VirtualClusterOptions{
			Name:            DefaultTestVclusterName,
			ServiceName:     DefaultTestVclusterServiceName,
			TargetNamespace: DefaultTestTargetNamespace,
		},
		Controllers:            controllercontext.ExistingControllers.Clone(),
		CurrentNamespace:       DefaultTestCurrentNamespace,
		CurrentNamespaceClient: pClient,
		VirtualManager:         newFakeManager(vClient),
		PhysicalManager:        newFakeManager(pClient),
	}
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
