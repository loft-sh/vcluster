package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var targetNamespace = "test"

func newFakeForwardSyncer(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *forwardController {
	err := vClient.IndexField(ctx, &corev1.Pod{}, constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
	if err != nil {
		panic(err)
	}

	return &forwardController{
		log:             loghelper.New("test-forwardcontroller"),
		synced:          func() {},
		targetNamespace: targetNamespace,
		virtualClient:   vClient,
		localClient:     pClient,
		target:          &generictesting.FakeSyncer{},
		scheme:          testingutil.NewScheme(),
	}
}
