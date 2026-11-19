package pods

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSyncToHostNodeSelectorWithNodeName(t *testing.T) {
	previousTranslator, previousName, previousServices := translate.Default, translate.VClusterName, specialservices.Default
	t.Cleanup(func() {
		translate.Default, translate.VClusterName, specialservices.Default = previousTranslator, previousName, previousServices
	})
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	translate.VClusterName = testingutil.DefaultTestVClusterName
	specialservices.Default = specialservices.NewDefaultServiceSyncer()
	dnsService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: translate.Default.HostName(nil, "kube-dns", "kube-system").Name, Namespace: testingutil.DefaultTestTargetNamespace},
		Spec:       corev1.ServiceSpec{ClusterIP: "2.2.2.2"},
	}

	vNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	vNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "infra-1", Labels: map[string]string{"pool": "infra"}}}
	pinned := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pinned", Namespace: "test"},
		Spec:       corev1.PodSpec{NodeName: "infra-1", Containers: []corev1.Container{{Name: "c", Image: "busybox"}}},
	}
	unpinned := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "unpinned", Namespace: "test"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "busybox"}}},
	}

	test := syncertesting.SyncTest{
		Name:                  "node selector applies with and without nodeName",
		InitialVirtualState:   []runtime.Object{vNamespace, vNode, pinned, unpinned},
		InitialPhysicalState:  []runtime.Object{pVclusterService.DeepCopy(), dnsService},
		ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
	}
	pClient, vClient, vConfig := test.Setup()
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	registerContext.Config.Networking.Advanced.ProxyKubelets.ByIP = false
	registerContext.Config.Sync.FromHost.Nodes.Enabled = true
	registerContext.Config.Sync.FromHost.Nodes.Selector.Labels = map[string]string{"pool": "tenant"}

	syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, New)
	for _, vPod := range []*corev1.Pod{pinned, unpinned} {
		_, err := syncer.(*podSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPod.DeepCopy()))
		assert.NilError(t, err)
	}

	hostPods := &corev1.PodList{}
	assert.NilError(t, pClient.List(context.Background(), hostPods, client.InNamespace(translate.Default.HostNamespace(nil, "test"))))
	byVirtualName := map[string]corev1.Pod{}
	for _, pPod := range hostPods.Items {
		byVirtualName[pPod.Annotations[translate.NameAnnotation]] = pPod
	}

	assert.Equal(t, byVirtualName["unpinned"].Spec.NodeName, "")
	assert.Equal(t, byVirtualName["unpinned"].Spec.NodeSelector["pool"], "tenant")
	assert.Equal(t, byVirtualName["pinned"].Spec.NodeName, "infra-1")
	assert.Equal(t, byVirtualName["pinned"].Spec.NodeSelector["pool"], "tenant")
}
