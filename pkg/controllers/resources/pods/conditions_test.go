package pods

import (
	"context"
	"fmt"
	"testing"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type conditionsTestCase struct {
	name string

	pPod *corev1.Pod
	vPod *corev1.Pod

	expectedPhysicalConditions []corev1.PodCondition
	expectedVirtualConditions  []corev1.PodCondition
}

func TestUpdateConditions(t *testing.T) {
	testCases := []conditionsTestCase{
		{
			name: "simple",

			pPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ptest",
					Namespace: "ptest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodScheduled,
							Status: "False",
							Reason: "my-reason",
						},
					},
				},
			},
			vPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vtest",
					Namespace: "vtest",
				},
			},

			expectedPhysicalConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
			},

			expectedVirtualConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
			},
		},
		{
			name: "keep-custom-vcondition",

			pPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ptest",
					Namespace: "ptest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodScheduled,
							Status: "False",
							Reason: "my-reason",
						},
					},
				},
			},

			vPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vtest",
					Namespace: "vtest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   "custom",
							Status: "True",
						},
					},
				},
			},

			expectedPhysicalConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
				{
					Type:   "custom",
					Status: "True",
				},
			},

			expectedVirtualConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
				{
					Type:   "custom",
					Status: "True",
				},
			},
		},
		{
			name: "dont-sync-custom-condition-up",

			pPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ptest",
					Namespace: "ptest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodScheduled,
							Status: "False",
							Reason: "my-reason",
						},
						{
							Type:   "custom",
							Status: "True",
						},
					},
				},
			},

			vPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vtest",
					Namespace: "vtest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{},
				},
			},

			expectedPhysicalConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
				{
					Type:   "custom",
					Status: "True",
				},
			},

			expectedVirtualConditions: []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: "False",
					Reason: "my-reason",
				},
			},
		},
		{
			name: "update-custom-condition",

			pPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ptest",
					Namespace: "ptest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   "custom",
							Status: "False",
							Reason: "my-reason",
						},
					},
				},
			},

			vPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vtest",
					Namespace: "vtest",
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   "custom",
							Status: "True",
						},
					},
				},
			},

			expectedPhysicalConditions: []corev1.PodCondition{
				{
					Type:   "custom",
					Status: "True",
				},
			},

			expectedVirtualConditions: []corev1.PodCondition{
				{
					Type:   "custom",
					Status: "True",
				},
			},
		},
	}

	for _, testCase := range testCases {
		fmt.Println(testCase.name)
		scheme := testingutil.NewScheme()
		ctx := context.Background()
		pClient := testingutil.NewFakeClient(scheme, testCase.pPod.DeepCopy())
		vClient := testingutil.NewFakeClient(scheme, testCase.vPod.DeepCopy())

		updated, err := UpdateConditions(&synccontext.SyncContext{
			Context:        ctx,
			Log:            loghelper.New(testCase.name),
			PhysicalClient: pClient,
			VirtualClient:  vClient,
		}, testCase.pPod, testCase.vPod)
		assert.NilError(t, err, "unexpected error in testCase %s", testCase.name)
		assert.DeepEqual(t, updated.Status.Conditions, testCase.expectedVirtualConditions)

		// check physical conditions
		newPod := &corev1.Pod{}
		err = pClient.Get(ctx, types.NamespacedName{Name: testCase.pPod.Name, Namespace: testCase.pPod.Namespace}, newPod)
		assert.NilError(t, err, "unexpected error while getting pPod in testCase %s", testCase.name)
		assert.DeepEqual(t, newPod.Status.Conditions, testCase.expectedPhysicalConditions)
	}
}
