package gatewaysync

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeleteReason(t *testing.T) {
	now := metav1.Now()
	deleting := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}}
	assert.Equal(t, DeleteReason(deleting), "virtual object was deleted by user")

	assert.Equal(t, DeleteReason(&corev1.ConfigMap{}), "host object was deleted")
	assert.Equal(t, DeleteReason(nil), "host object was deleted")
}
