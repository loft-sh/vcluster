package backup

import (
	"context"
	"testing"

	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v5/pkg/apis/storage/v1"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

// sharedSecret builds a shared secret as unstructured: the api module no longer needs to carry
// the SharedSecret type for a backup of a 4.x platform to keep them.
func sharedSecret(namespace, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(sharedSecretGVK)
	u.SetNamespace(namespace)
	u.SetName(name)
	u.SetUID("uid")
	u.SetResourceVersion("1")
	u.Object["spec"] = map[string]interface{}{"data": map[string]interface{}{"key": "dmFsdWU="}}
	u.Object["status"] = map[string]interface{}{"phase": "Ready"}
	return u
}

// A 4.x platform still serves shared secrets, so a newer CLI backing it up keeps them. The scheme
// deliberately leaves out the storage types, as it will once the api module drops SharedSecret.
func TestSharedSecretsOnAPlatformThatServesThem(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(sharedSecretGVK, meta.RESTScopeNamespace)
	c := fake.NewClientBuilder().WithScheme(scheme).WithRESTMapper(mapper).WithObjects(sharedSecret("loft", "registry")).Build()

	objs, err := sharedSecrets(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 1 {
		t.Fatalf("expected 1 shared secret, got %d", len(objs))
	}
	got := objs[0].(*unstructured.Unstructured)
	if got.GroupVersionKind() != sharedSecretGVK || got.GetName() != "registry" || got.GetNamespace() != "loft" {
		t.Errorf("wrong object: %v %s/%s", got.GroupVersionKind(), got.GetNamespace(), got.GetName())
	}
	if got.GetUID() != "" || got.GetResourceVersion() != "" {
		t.Errorf("metadata not reset: uid %q resourceVersion %q", got.GetUID(), got.GetResourceVersion())
	}
	if _, found := got.Object["status"]; found {
		t.Errorf("status should be cleared: %v", got.Object["status"])
	}
	if data, _, _ := unstructured.NestedString(got.Object, "spec", "data", "key"); data != "dmFsdWU=" {
		t.Errorf("spec data not kept: %q", data)
	}
}

// A 5.0 platform no longer serves shared secrets, so the backup skips them without an error.
func TestSharedSecretsOnAPlatformThatDoesNotServeThem(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithRESTMapper(meta.NewDefaultRESTMapper(nil)).Build()

	objs, err := sharedSecrets(context.Background(), c)
	if err != nil {
		t.Fatalf("expected no error when shared secrets are not served, got %v", err)
	}
	if len(objs) != 0 {
		t.Fatalf("expected no shared secrets, got %d", len(objs))
	}
}

// A backup of a 4.x platform holds shared secrets, which a 5.0 platform cannot restore, so the
// CLI notes it. A backup without them must not trigger the note.
func TestIncludesSharedSecrets(t *testing.T) {
	if IncludesSharedSecrets([]runtime.Object{secret("loft", "loft-cert")}) {
		t.Error("a backup without shared secrets reported them")
	}
	if !IncludesSharedSecrets([]runtime.Object{secret("loft", "loft-cert"), sharedSecret("loft", "registry")}) {
		t.Error("a backup with a shared secret did not report it")
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
