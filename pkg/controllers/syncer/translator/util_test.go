package translator

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewIfNil(t *testing.T) {
	examplePod := &corev1.Pod{}
	newPod := NewIfNil(examplePod, examplePod)
	assert.Equal(t, examplePod == newPod, true)

	var updatedPod *corev1.Pod
	otherPod := NewIfNil(updatedPod, examplePod)
	assert.Equal(t, otherPod != examplePod, true)
}
