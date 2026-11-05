package pods

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources the host applied on its own, for example LimitRange defaults, must not be
// resized back to the virtual values (issue #4091).
func TestSyncDoesNotResizeHostAppliedResources(t *testing.T) {
	f := newConflictFixture(t)
	f.syncer.hostClusterVersion = utilversion.MustParseSemantic("1.35.0")

	pNew := f.host()
	pNew.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")},
	}
	f.update(f.pClient, pNew)
	pNew = f.host()
	pNew.Status.PodIP = "10.0.0.1"
	pNew.Status.Phase = corev1.PodRunning
	f.updateStatus(f.pClient, pNew)

	resize := &resizeRecordingClient{Client: f.syncCtx.HostClient}
	f.syncCtx.HostClient = resize
	assert.NilError(t, f.sync(f.host(), f.virtual()))

	assert.Equal(t, resize.calls, 0, "host resources were resized although the virtual pod did not change")
	assert.Equal(t, f.virtual().Status.Phase, corev1.PodRunning, "host status did not reach the virtual pod")
}

// When the host denies the resize, the virtual pod must still receive the host status
// and the sync must not keep failing on it.
func TestSyncKeepsSyncingStatusWhenResizeIsForbidden(t *testing.T) {
	f := newConflictFixture(t)
	f.syncer.hostClusterVersion = utilversion.MustParseSemantic("1.35.0")

	vNew := f.virtual()
	vNew.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")},
	}
	f.update(f.vClient, vNew)

	pNew := f.host()
	pNew.Status.PodIP = "10.0.0.1"
	pNew.Status.Phase = corev1.PodRunning
	f.updateStatus(f.pClient, pNew)

	resize := &resizeRecordingClient{Client: f.syncCtx.HostClient, forbidden: true}
	f.syncCtx.HostClient = resize
	assert.NilError(t, f.sync(f.host(), f.virtual()))

	assert.Equal(t, resize.calls, 1, "the virtual resource change was not sent to the host")
	assert.Equal(t, f.virtual().Status.Phase, corev1.PodRunning, "host status did not reach the virtual pod")
	assert.Equal(t, f.virtual().Status.PodIP, "10.0.0.1")
}

// resizeRecordingClient counts resize subresource patches and optionally denies them,
// like a host without the pods/resize RBAC rule.
type resizeRecordingClient struct {
	client.Client

	calls     int
	forbidden bool
}

func (c *resizeRecordingClient) SubResource(subResource string) client.SubResourceClient {
	if subResource != "resize" {
		return c.Client.SubResource(subResource)
	}

	return &resizeSubResourceClient{SubResourceClient: c.Client.SubResource(subResource), parent: c}
}

type resizeSubResourceClient struct {
	client.SubResourceClient

	parent *resizeRecordingClient
}

func (c *resizeSubResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	c.parent.calls++
	if c.parent.forbidden {
		return kerrors.NewForbidden(schema.GroupResource{Resource: "pods"}, obj.GetName(), errors.New("cannot patch resource \"pods/resize\""))
	}

	return c.SubResourceClient.Patch(ctx, obj, patch, opts...)
}
