package priorityclasses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSyncToHost(t *testing.T) {
	vObj := schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Value:      1,
	}
	pObj := schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{Name: "vcluster-stuff-x-test-x-suffix"},

		Value: 1,
	}

	pObj.Annotations = translate.HostAnnotations(&vObj, &pObj)
	testCases := []struct {
		name         string
		syncToHost   bool
		syncFromHost bool
		isDelete     bool
		pObjExists   bool
		expectPObj   bool
		expectVObj   bool
		withVObj     bool
	}{
		{
			name:       "sync to host",
			syncToHost: true,
			withVObj:   true,
			expectPObj: true,
		},
		{
			name:       "sync to host create",
			syncToHost: true,
			isDelete:   false,
			pObjExists: false,
			expectPObj: true,
		},
		{
			name:         "sync to host delete virtual",
			isDelete:     true,
			withVObj:     true,
			expectVObj:   false,
			syncFromHost: true,
		},
		{
			name:         "2 way sync delete virtual",
			isDelete:     true,
			expectVObj:   false,
			syncFromHost: true,
			syncToHost:   true,
		},
		{
			name:         "2 way sync create physical",
			isDelete:     false,
			expectVObj:   true,
			withVObj:     true,
			expectPObj:   true,
			syncFromHost: true,
			syncToHost:   true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			initialVirtualObjects := []runtime.Object{}
			if tC.withVObj {
				initialVirtualObjects = append(initialVirtualObjects, vObj.DeepCopy())
			}
			expectedVirtualObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectVObj {
				expectedVirtualObjects[schedulingv1.SchemeGroupVersion.WithKind("PriorityClass")] = []runtime.Object{vObj.DeepCopy()}
			}
			initialPhysicalObjects := []runtime.Object{}
			if tC.pObjExists {
				initialPhysicalObjects = append(initialPhysicalObjects, pObj.DeepCopy())
			}
			expectedPhysicalObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectPObj {
				expectedPhysicalObjects[schedulingv1.SchemeGroupVersion.WithKind("PriorityClass")] = []runtime.Object{pObj.DeepCopy()}
			}
			test := syncertesting.SyncTest{
				Name:                  tC.name,
				InitialVirtualState:   initialVirtualObjects,
				InitialPhysicalState:  initialPhysicalObjects,
				ExpectedVirtualState:  expectedVirtualObjects,
				ExpectedPhysicalState: expectedPhysicalObjects,
			}
			// setting up the clients
			pClient, vClient, vConfig := test.Setup()
			vConfig.Sync.FromHost.PriorityClasses.Enabled = tC.syncFromHost
			vConfig.Sync.ToHost.PriorityClasses.Enabled = tC.syncToHost
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)

			syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, New)
			event := synccontext.NewSyncToHostEvent(vObj.DeepCopy())
			if tC.isDelete {
				event.Type = synccontext.SyncEventTypeDelete
			}
			_, err := syncer.(*priorityClassSyncer).SyncToHost(syncCtx, event)
			assert.NilError(t, err)

			test.Validate(t)
		})
	}
}

func TestSyncToVirtual(t *testing.T) {
	vObj := schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Value:      1,
	}
	pObj := schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},

		Value: 1,
	}

	testCases := []struct {
		name         string
		syncToHost   bool
		syncFromHost bool
		isDelete     bool
		withPObj     bool
		expectPObj   bool
		expectVObj   bool
		withVObj     bool
	}{
		{
			name:         "sync to virtual",
			syncFromHost: true,
			withVObj:     false,
			expectVObj:   true,
			withPObj:     true,
			expectPObj:   true,
		},
		{
			name:       "sync to virtual tohost",
			syncToHost: true,
			withVObj:   false,
			expectVObj: false,
			withPObj:   true,
			expectPObj: false,
		},
		{
			name:         "sync to virtual 2 way",
			syncFromHost: true,
			syncToHost:   true,
			isDelete:     false,
			withVObj:     false,
			expectVObj:   true,
			withPObj:     true,
			expectPObj:   true,
		},
		{
			name:         "sync to virtual 2 way delete",
			syncFromHost: true,
			syncToHost:   true,
			isDelete:     true,
			withVObj:     false,
			expectVObj:   false,
			withPObj:     true,
			expectPObj:   false,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			initialVirtualObjects := []runtime.Object{}
			if tC.withVObj {
				initialVirtualObjects = append(initialVirtualObjects, vObj.DeepCopy())
			}
			expectedVirtualObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectVObj {
				expectedVirtualObjects[schedulingv1.SchemeGroupVersion.WithKind("PriorityClass")] = []runtime.Object{vObj.DeepCopy()}
			}
			initialPhysicalObjects := []runtime.Object{}
			if tC.withPObj {
				initialPhysicalObjects = append(initialPhysicalObjects, pObj.DeepCopy())
			}
			expectedPhysicalObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectPObj {
				expectedPhysicalObjects[schedulingv1.SchemeGroupVersion.WithKind("PriorityClass")] = []runtime.Object{pObj.DeepCopy()}
			}
			test := syncertesting.SyncTest{
				Name:                  tC.name,
				InitialVirtualState:   initialVirtualObjects,
				InitialPhysicalState:  initialPhysicalObjects,
				ExpectedVirtualState:  expectedVirtualObjects,
				ExpectedPhysicalState: expectedPhysicalObjects,
			}
			// setting up the clients
			pClient, vClient, vConfig := test.Setup()
			vConfig.Sync.FromHost.PriorityClasses.Enabled = tC.syncFromHost
			vConfig.Sync.ToHost.PriorityClasses.Enabled = tC.syncToHost
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)

			syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, New)
			event := synccontext.NewSyncToVirtualEvent(pObj.DeepCopy())
			if tC.isDelete {
				event.Type = synccontext.SyncEventTypeDelete
			}
			_, err := syncer.(*priorityClassSyncer).SyncToVirtual(syncCtx, event)
			assert.NilError(t, err)

			test.Validate(t)
		})
	}
}
