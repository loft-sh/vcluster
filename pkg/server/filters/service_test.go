package filters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type denyUpdateClient struct {
	client.Client
}

func (d *denyUpdateClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	return kerrors.NewForbidden(schema.GroupResource{Resource: "services"}, obj.GetName(), fmt.Errorf("update denied"))
}

func TestUpdateServiceKeepsHostServiceWhenVirtualUpdateDenied(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	registerCtx := syncertesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
	syncCtx := registerCtx.ToSyncContext("update-service")

	oldVService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "ext",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "example.com",
		},
	}
	hostName := mappings.VirtualToHost(syncCtx, oldVService.Name, oldVService.Namespace, mappings.Services())
	pService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostName.Name,
			Namespace: hostName.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "example.com",
		},
	}
	if err := pClient.Create(context.Background(), pService); err != nil {
		t.Fatalf("create host service: %v", err)
	}

	newVService := oldVService.DeepCopy()
	newVService.Spec.Type = corev1.ServiceTypeClusterIP
	newVService.Spec.ExternalName = ""
	newVService.Spec.ClusterIP = ""
	body, err := json.Marshal(newVService)
	if err != nil {
		t.Fatalf("marshal service: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/namespaces/default/services/ext", bytes.NewReader(body))
	syncCtx.VirtualClient = &denyUpdateClient{Client: vClient}
	syncCtx.HostClient = pClient

	_, err = updateService(syncCtx, req, encoding.NewDecoder(scheme.Scheme, false), oldVService)
	if !kerrors.IsForbidden(err) {
		t.Fatalf("expected forbidden virtual update, got %v", err)
	}

	got := &corev1.Service{}
	err = pClient.Get(context.Background(), hostName, got)
	if kerrors.IsNotFound(err) {
		t.Fatal("host service was deleted after a denied virtual update")
	}
	if err != nil {
		t.Fatalf("get host service: %v", err)
	}
	if got.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf("expected host service type ClusterIP after patch, got %s", got.Spec.Type)
	}
}
