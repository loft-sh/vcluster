package testing

import (
	"context"
	"testing"

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

	// run migrate
	mapper, ok := object.(synccontext.Mapper)
	if ok {
		err := mapper.Migrate(ctx, mapper)
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
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	registerCtx := &synccontext.RegisterContext{
		Context:                ctx,
		Config:                 vConfig,
		CurrentNamespace:       testingutil.DefaultTestCurrentNamespace,
		CurrentNamespaceClient: pClient,
		VirtualManager:         testingutil.NewFakeManager(vClient),
		PhysicalManager:        testingutil.NewFakeManager(pClient),
		Mappings:               mappings.NewMappingsRegistry(mappingsStore),
	}

	// make sure we do not ensure any CRDs
	util.EnsureCRD = func(_ context.Context, _ *rest.Config, _ []byte, _ schema.GroupVersionKind) error {
		return nil
	}

	// register & migrate mappers
	resources.MustRegisterMappings(registerCtx)
	for _, mapper := range registerCtx.Mappings.List() {
		err := mapper.Migrate(registerCtx, mapper)
		if err != nil {
			panic(err)
		}
	}

	return registerCtx
}
