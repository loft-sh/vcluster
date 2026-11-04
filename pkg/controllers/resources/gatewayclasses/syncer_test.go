package gatewayclasses

import (
	"context"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGatewayClassSpecToVirtualRemovesParametersRefAndPreservesTenantVisibleFields(t *testing.T) {
	description := "tenant visible class"
	hostSpec := gatewayv1.GatewayClassSpec{
		ControllerName: "example.com/gateway-controller",
		Description:    &description,
		ParametersRef:  gatewayClassParametersRef("host-config", "host-only"),
	}

	got := gatewayClassSpecToVirtual(hostSpec)
	if got.ParametersRef != nil {
		t.Fatalf("expected parametersRef to be removed, got %#v", got.ParametersRef)
	}
	if got.ControllerName != hostSpec.ControllerName {
		t.Fatalf("expected controllerName %q, got %q", hostSpec.ControllerName, got.ControllerName)
	}
	if got.Description == nil || *got.Description != description {
		t.Fatalf("expected description %q, got %#v", description, got.Description)
	}
}

func TestGatewayClassSyncToVirtualSanitizesParametersRefAfterPatches(t *testing.T) {
	withGatewayClassPatchThatAddsParametersRef(t)
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.GatewayClasses.Enabled = true
	vcConfig.Sync.FromHost.GatewayClasses.Patches = []rootconfig.TranslatePatch{{Path: "spec.parametersRef.name", Expression: "'patched-config'"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, New)
	syncer := object.(*gatewayClassSyncer)

	host := gatewayClass("tenant-class")
	_, err := syncer.SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(host))
	if err != nil {
		t.Fatalf("sync to virtual: %v", err)
	}

	got := &gatewayv1.GatewayClass{}
	if err := vClient.Get(context.Background(), types.NamespacedName{Name: host.Name}, got); err != nil {
		t.Fatalf("expected virtual GatewayClass to be created: %v", err)
	}
	if got.Spec.ParametersRef != nil {
		t.Fatalf("expected parametersRef to remain sanitized after create patches, got %#v", got.Spec.ParametersRef)
	}
}

func TestGatewayClassSyncSanitizesParametersRefAfterPatches(t *testing.T) {
	withGatewayClassPatchThatAddsParametersRef(t)
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.GatewayClasses.Enabled = true
	vcConfig.Sync.FromHost.GatewayClasses.Patches = []rootconfig.TranslatePatch{{Path: "spec.parametersRef.name", Expression: "'patched-config'"}}
	host := gatewayClass("tenant-class")
	virtual := &gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: host.Name}, Spec: gatewayv1.GatewayClassSpec{ControllerName: "old.example.com/controller"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, host)
	vClient := testingutil.NewFakeClient(scheme.Scheme, virtual.DeepCopy())
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, New)
	syncer := object.(*gatewayClassSyncer)
	currentVirtual := &gatewayv1.GatewayClass{}
	if err := vClient.Get(context.Background(), types.NamespacedName{Name: host.Name}, currentVirtual); err != nil {
		t.Fatalf("get current virtual GatewayClass: %v", err)
	}

	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEvent(host, currentVirtual))
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	got := &gatewayv1.GatewayClass{}
	if err := vClient.Get(context.Background(), types.NamespacedName{Name: host.Name}, got); err != nil {
		t.Fatalf("expected virtual GatewayClass to be updated: %v", err)
	}
	if got.Spec.ParametersRef != nil {
		t.Fatalf("expected parametersRef to remain sanitized after update patches, got %#v", got.Spec.ParametersRef)
	}
}

func withGatewayClassPatchThatAddsParametersRef(t *testing.T) {
	t.Helper()
	origVirtual := pro.ApplyPatchesVirtualObject
	origHost := pro.ApplyPatchesHostObject
	patchFn := func(_ *synccontext.SyncContext, _, obj, _ client.Object, _ []rootconfig.TranslatePatch, _ bool) error {
		gwClass, ok := obj.(*gatewayv1.GatewayClass)
		if ok {
			gwClass.Spec.ParametersRef = gatewayClassParametersRef("patched-config", "patched-namespace")
		}
		return nil
	}
	pro.ApplyPatchesVirtualObject = patchFn
	pro.ApplyPatchesHostObject = patchFn
	t.Cleanup(func() {
		pro.ApplyPatchesVirtualObject = origVirtual
		pro.ApplyPatchesHostObject = origHost
	})
}

func gatewayClass(name string) *gatewayv1.GatewayClass {
	description := "tenant visible class"
	return &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "example.com/gateway-controller",
			Description:    &description,
			ParametersRef:  gatewayClassParametersRef(name+"-config", "host-only"),
		},
	}
}

func gatewayClassParametersRef(name string, namespace string) *gatewayv1.ParametersReference {
	ns := gatewayv1.Namespace(namespace)
	return &gatewayv1.ParametersReference{
		Group:     gatewayv1.Group("example.com"),
		Kind:      gatewayv1.Kind("GatewayClassConfig"),
		Name:      name,
		Namespace: &ns,
	}
}

var _ runtime.Object = &gatewayv1.GatewayClass{}
