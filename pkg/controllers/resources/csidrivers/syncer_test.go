package csidrivers

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const kind = "CSIDriver"

var (
	fsGroupPolicyFile = storagev1.FileFSGroupPolicy
)

func TestSync(t *testing.T) {
	pObjectMeta := metav1.ObjectMeta{
		Name: "test-csidriver",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:            "test-csidriver",
		ResourceVersion: "999",
	}

	pObj := &storagev1.CSIDriver{
		ObjectMeta: pObjectMeta,
		Spec: storagev1.CSIDriverSpec{
			AttachRequired:       boolRef(true),
			PodInfoOnMount:       boolRef(false),
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{storagev1.VolumeLifecycleEphemeral, storagev1.VolumeLifecyclePersistent},
			StorageCapacity:      boolRef(true),
			FSGroupPolicy:        &fsGroupPolicyFile,
			TokenRequests: []storagev1.TokenRequest{
				{Audience: "foo", ExpirationSeconds: int64Ref(120)},
			},
			RequiresRepublish: boolRef(true),
			SELinuxMount:      boolRef(true),
		},
	}

	vObj := &storagev1.CSIDriver{
		ObjectMeta: vObjectMeta,
		Spec: storagev1.CSIDriverSpec{
			AttachRequired:       boolRef(true),
			PodInfoOnMount:       boolRef(false),
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{storagev1.VolumeLifecycleEphemeral, storagev1.VolumeLifecyclePersistent},
			StorageCapacity:      boolRef(true),
			FSGroupPolicy:        &fsGroupPolicyFile,
			TokenRequests: []storagev1.TokenRequest{
				{Audience: "foo", ExpirationSeconds: int64Ref(120)},
			},
			RequiresRepublish: boolRef(true),
			SELinuxMount:      boolRef(true),
		},
	}

	pObjUpdated := &storagev1.CSIDriver{
		ObjectMeta: pObjectMeta,
		Spec: storagev1.CSIDriverSpec{
			AttachRequired:       boolRef(false),
			PodInfoOnMount:       boolRef(true),
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{storagev1.VolumeLifecycleEphemeral, storagev1.VolumeLifecyclePersistent},
			StorageCapacity:      boolRef(false),
			TokenRequests: []storagev1.TokenRequest{
				{Audience: "bar", ExpirationSeconds: int64Ref(120)},
				{Audience: "baz", ExpirationSeconds: int64Ref(60)},
			},
			RequiresRepublish: boolRef(true),
			SELinuxMount:      boolRef(false),
		},
	}

	vObjUpdated := &storagev1.CSIDriver{
		ObjectMeta: pObjectMeta,
		Spec: storagev1.CSIDriverSpec{
			AttachRequired:       boolRef(false),
			PodInfoOnMount:       boolRef(true),
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{storagev1.VolumeLifecycleEphemeral, storagev1.VolumeLifecyclePersistent},
			StorageCapacity:      boolRef(false),
			TokenRequests: []storagev1.TokenRequest{
				{Audience: "bar", ExpirationSeconds: int64Ref(120)},
				{Audience: "baz", ExpirationSeconds: int64Ref(60)},
			},
			RequiresRepublish: boolRef(true),
			SELinuxMount:      boolRef(false),
		},
	}

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.FromHost.CSIDrivers.Enabled = "true"
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csidriverSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csidriverSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csidriverSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated, vObj))
				assert.NilError(t, err)
			},
		},
	})
}

func int64Ref(i int64) *int64 {
	return &i
}

func boolRef(b bool) *bool {
	return &b
}
