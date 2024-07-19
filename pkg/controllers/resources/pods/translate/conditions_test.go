package translate

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		updateConditions(testCase.pPod, testCase.vPod, testCase.vPod.Status.DeepCopy())
		assert.DeepEqual(t, testCase.vPod.Status.Conditions, testCase.expectedVirtualConditions)
		assert.DeepEqual(t, testCase.pPod.Status.Conditions, testCase.expectedPhysicalConditions)
	}
}
