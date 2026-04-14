package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type translateKubeletSubPathTestCase struct {
	name     string
	vPods    []corev1.Pod
	pPods    []corev1.Pod
	input    string
	expected string
}

func TestTranslateKubeletSubPath(t *testing.T) {
	const (
		vPodName      = "my-pod"
		vPodNamespace = "my-ns"
		vUID          = types.UID("aaaa-bbbb-cccc-virtual")
		pUID          = types.UID("dddd-eeee-ffff-physical")
	)

	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	pPodName := translate.Default.HostName(nil, vPodName, vPodNamespace).Name
	pPodNS := testingutil.DefaultTestTargetNamespace

	testCases := []translateKubeletSubPathTestCase{
		{
			name: "rewrites virtual UID to physical UID in deep sub-path",
			vPods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: vPodName, Namespace: vPodNamespace, UID: vUID}},
			},
			pPods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: pPodName, Namespace: pPodNS, UID: pUID}},
			},
			input:    KubeletPodPath + "/" + string(vUID) + "/volumes/kubernetes.io~csi/pvc-abc/mount",
			expected: KubeletPodPath + "/" + string(pUID) + "/volumes/kubernetes.io~csi/pvc-abc/mount",
		},
		{
			name: "rewrites virtual UID with no sub-path under UID",
			vPods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: vPodName, Namespace: vPodNamespace, UID: vUID}},
			},
			pPods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: pPodName, Namespace: pPodNS, UID: pUID}},
			},
			input:    KubeletPodPath + "/" + string(vUID),
			expected: KubeletPodPath + "/" + string(pUID),
		},
		{
			name:     "returns empty string for unknown UID",
			input:    KubeletPodPath + "/unknown-uid-that-does-not-exist/volumes/kubernetes.io~csi/pvc-abc/mount",
			expected: "",
		},
		{
			name:     "returns empty string for empty UID segment",
			input:    KubeletPodPath + "/",
			expected: "",
		},
		{
			name: "returns empty string when physical pod is not found",
			vPods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "orphan-pod", Namespace: vPodNamespace, UID: "orphan-virtual-uid"}},
			},
			// pPods intentionally empty — physical pod never created
			input:    KubeletPodPath + "/orphan-virtual-uid/volumes/kubernetes.io~csi/pvc-abc/mount",
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pRuntimeObjs := make([]runtime.Object, len(testCase.pPods))
			for i := range testCase.pPods {
				pRuntimeObjs[i] = &testCase.pPods[i]
			}
			vRuntimeObjs := make([]runtime.Object, len(testCase.vPods))
			for i := range testCase.vPods {
				vRuntimeObjs[i] = &testCase.vPods[i]
			}

			pClient := testingutil.NewFakeClient(scheme.Scheme, pRuntimeObjs...)
			vClient := testingutil.NewFakeClient(scheme.Scheme, vRuntimeObjs...)
			registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)

			tr := &translator{
				log:     loghelper.New("test"),
				vClient: vClient,
				pClient: pClient,
			}

			result := tr.translateKubeletSubPath(registerCtx.ToSyncContext("test"), testCase.input)
			assert.Equal(t, result, testCase.expected)
		})
	}
}
