package testing

import (
	"context"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"github.com/loft-sh/vcluster/pkg/util/log"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"

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
	DefaultTestVClusterName        = "vcluster"
	DefaultTestVClusterServiceName = "vcluster"
)

func FakeStartSyncer(t *testing.T, ctx *synccontext.RegisterContext, create func(ctx *synccontext.RegisterContext) (syncer.Object, error)) (*synccontext.SyncContext, syncer.Object) {
	object, err := create(ctx)
	assert.NilError(t, err)
	if object == nil {
		t.Fatal("object is nil")
	}

	// run register indices
	registerer, ok := object.(syncer.IndicesRegisterer)
	if ok {
		err := registerer.RegisterIndices(ctx)
		assert.NilError(t, err)
	}

	syncCtx := ctx.ToSyncContext(object.Name())
	syncCtx.Log = loghelper.NewFromExisting(log.NewLog(0), object.Name())
	return syncCtx, object
}

func NewFakeRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	ctx := context.Background()
	mappingsStore, _ := store.NewStore(ctx, vClient, pClient, store.NewMemoryBackend())

	// create register context
	translate.Default = translate.NewSingleNamespaceTranslator(DefaultTestTargetNamespace)
	registerCtx := &synccontext.RegisterContext{
		Context:                ctx,
		Config:                 vConfig,
		CurrentNamespace:       DefaultTestCurrentNamespace,
		CurrentNamespaceClient: pClient,
		VirtualManager:         newFakeManager(vClient),
		PhysicalManager:        newFakeManager(pClient),
		Mappings:               mappings.NewMappingsRegistry(mappingsStore),
	}

	// make sure we do not ensure any CRDs
	util.EnsureCRD = func(_ context.Context, _ *rest.Config, _ []byte, _ schema.GroupVersionKind) error {
		return nil
	}

	resources.MustRegisterMappings(registerCtx)
	return registerCtx
}

func NewFakeConfig() *config.VirtualClusterConfig {
	// default config
	defaultConfig, err := vclusterconfig.NewDefaultConfig()
	if err != nil {
		panic(err.Error())
	}

	// parse config
	vConfig := &config.VirtualClusterConfig{
		Config:                  *defaultConfig,
		Name:                    DefaultTestVClusterName,
		ControlPlaneService:     DefaultTestVClusterName,
		WorkloadService:         DefaultTestVClusterServiceName,
		WorkloadNamespace:       DefaultTestTargetNamespace,
		WorkloadTargetNamespace: DefaultTestTargetNamespace,
	}

	err = config.ValidateConfigAndSetDefaults(vConfig)
	if err != nil {
		panic(err.Error())
	}

	return vConfig
}

type fakeEventBroadcaster struct{}

func (f *fakeEventBroadcaster) StartEventWatcher(_ func(*corev1.Event)) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartRecordingToSink(_ record.EventSink) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartLogging(_ func(format string, args ...interface{})) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartStructuredLogging(_ klog.Level) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) NewRecorder(_ *runtime.Scheme, _ corev1.EventSource) record.EventRecorder {
	return f
}

func (f *fakeEventBroadcaster) Shutdown() {}

func (f *fakeEventBroadcaster) Event(_ runtime.Object, _, _, _ string) {}

func (f *fakeEventBroadcaster) Eventf(_ runtime.Object, _, _, _ string, _ ...interface{}) {
}

func (f *fakeEventBroadcaster) AnnotatedEventf(_ runtime.Object, _ map[string]string, _, _, _ string, _ ...interface{}) {
}
