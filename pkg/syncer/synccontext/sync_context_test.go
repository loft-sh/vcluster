package synccontext

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestWithoutMappingContextPreservesContextAndRemovesMapping(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	baseCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	nameMapping := NameMapping{
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
		VirtualName:      types.NamespacedName{Name: "secret", Namespace: "default"},
		HostName:         types.NamespacedName{Name: "secret-x-vcluster", Namespace: "host"},
	}
	syncCtx := &SyncContext{
		Context: WithMapping(baseCtx, nameMapping),
	}

	strippedCtx := WithoutMapping(syncCtx)
	_, ok := MappingFrom(strippedCtx)
	assert.Assert(t, !ok)

	originalMapping, ok := MappingFrom(syncCtx)
	assert.Assert(t, ok)
	assert.DeepEqual(t, originalMapping, nameMapping)

	strippedDeadline, ok := strippedCtx.Deadline()
	assert.Assert(t, ok)
	assert.Assert(t, strippedDeadline.Equal(deadline))

	cancel()
	select {
	case <-strippedCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("stripped context did not observe cancellation")
	}
}
