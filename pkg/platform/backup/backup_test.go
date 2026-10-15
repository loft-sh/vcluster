package backup

import (
	"context"
	"testing"

	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v5/pkg/apis/storage/v1"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := storagev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func secret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, UID: "uid", ResourceVersion: "1"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"key": []byte(name)},
	}
}

func TestPlatformSecrets(t *testing.T) {
	scheme := testScheme(t)
	// loft-license-cache deliberately missing: a fresh platform may not have refreshed yet
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		secret("loft", "loft-cert"),
		secret("loft", "loft-network-control-key"),
		secret("other", "loft-cert"),
	).Build()

	objs, err := platformSecrets(context.Background(), c, "loft")
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(objs))
	}
	for _, o := range objs {
		s := o.(*corev1.Secret)
		if s.Namespace != "loft" || s.ResourceVersion != "" || s.UID != "" {
			t.Errorf("secret %s: metadata not reset or wrong namespace: %+v", s.Name, s.ObjectMeta)
		}
	}

	if _, err := platformSecrets(context.Background(), c, ""); err == nil {
		t.Error("expected an error for an empty namespace")
	}
}

func TestAllSkipsPlatformSecrets(t *testing.T) {
	scheme := testScheme(t)
	projectutil.SetProjectNamespacePrefix("p-")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret("loft", "loft-cert")).Build()

	count := func(skip []string) int {
		objs, errs := All(context.Background(), c, "loft", skip, func(string) {})
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		n := 0
		for _, o := range objs {
			if _, ok := o.(*corev1.Secret); ok {
				n++
			}
		}
		return n
	}

	if got := count(nil); got != 1 {
		t.Errorf("expected the platform secret in the backup, got %d secrets", got)
	}
	if got := count([]string{"platformsecrets"}); got != 0 {
		t.Errorf("expected no secrets with --skip platformsecrets, got %d", got)
	}
}

func TestVirtualClusterInstancesKeepServiceUID(t *testing.T) {
	scheme := testScheme(t)
	projectutil.SetProjectNamespacePrefix("p-")
	vci := &storagev1.VirtualClusterInstance{
		ObjectMeta: metav1.ObjectMeta{Namespace: "p-production", Name: "runpod", UID: "uid", ResourceVersion: "7"},
		Status: storagev1.VirtualClusterInstanceStatus{
			ServiceUID: "a51c64b2-9ebf-4c72-b849-2caba5e0ba8d",
			Phase:      storagev1.InstanceReady,
			Conditions: agentstoragev1.Conditions{{Type: agentstoragev1.ConditionType("Ready"), Status: corev1.ConditionTrue}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vci).Build()

	objs, err := virtualClusterInstances(context.Background(), c, []string{"production"})
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(objs))
	}
	got := objs[0].(*storagev1.VirtualClusterInstance)
	if got.Status.ServiceUID != vci.Status.ServiceUID {
		t.Errorf("serviceUID not kept: %q", got.Status.ServiceUID)
	}
	if got.Status.Phase != "" || len(got.Status.Conditions) != 0 {
		t.Errorf("rest of the status should be cleared: %+v", got.Status)
	}
	if got.ResourceVersion != "" || got.UID != "" {
		t.Errorf("metadata not reset: %+v", got.ObjectMeta)
	}
}
