package testing

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/util/log"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
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
		Context:                context.Background(),
		Config:                 NewFakeConfig(),
		CurrentNamespace:       DefaultTestCurrentNamespace,
		CurrentNamespaceClient: pClient,
		VirtualManager:         newFakeManager(vClient),
		PhysicalManager:        newFakeManager(pClient),
	}
}

func NewFakeConfig() *config.VirtualClusterConfig {
	// find vCluster config
	workDir, err := os.Getwd()
	if err != nil {
		panic("current workDir: " + err.Error())
	}

	// find base dir
	configPath := ""
	for {
		configPath = filepath.Join(workDir, "chart", "values.yaml")
		_, err = os.Stat(configPath)
		if err == nil {
			break
		} else if workDir == "/" {
			panic("couldn't find chart/values.yaml")
		}

		workDir = filepath.Dir(workDir)
	}

	// parse config
	vConfig, err := config.ParseConfig(configPath, DefaultTestVClusterName, nil)
	if err != nil {
		panic("load test config: " + workDir + " - " + err.Error())
	}

	vConfig.Name = DefaultTestVClusterName
	vConfig.WorkloadService = DefaultTestVClusterServiceName
	vConfig.WorkloadNamespace = DefaultTestTargetNamespace
	vConfig.WorkloadTargetNamespace = DefaultTestTargetNamespace
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
