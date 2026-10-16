package kubeconfig

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPatchKubeConfigSecretPreservesUnmanagedFieldsAndDoesNotCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "export", Namespace: "central", Labels: map[string]string{"owner": "gitops"}},
		Data:       map[string][]byte{"external-note": []byte("keep"), CertificateKeySecretKey: []byte("stale-key")},
	}
	created, patched := 0, 0
	host := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).WithInterceptorFuncs(interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			created++
			return c.Create(ctx, obj, opts...)
		},
		Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
			patched++
			return c.Patch(ctx, obj, patch, opts...)
		},
	}).Build()
	cfg := &clientcmdapi.Config{
		CurrentContext: "export",
		Contexts:       map[string]*clientcmdapi.Context{"export": {Cluster: "virtual", AuthInfo: "runtime"}},
		Clusters:       map[string]*clientcmdapi.Cluster{"virtual": {Server: "https://virtual.example.com"}},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{"runtime": {Token: "issued-token"}},
	}
	ctx := context.Background()
	for range 2 {
		if err := PatchKubeConfigSecret(ctx, host, secret.Name, secret.Namespace, cfg); err != nil {
			t.Fatal(err)
		}
	}
	if err := host.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		t.Fatal(err)
	}
	if patched != 1 || string(secret.Data[TokenSecretKey]) != "issued-token" || len(secret.Data[KubeconfigSecretKey]) == 0 || len(secret.Data[CertificateKeySecretKey]) != 0 {
		t.Fatal("repair did not restore credentials idempotently or remove the stale key")
	}
	if string(secret.Data["external-note"]) != "keep" || secret.Labels["owner"] != "gitops" {
		t.Fatal("repair changed unmanaged data or metadata")
	}
	if err := host.Delete(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if err := PatchKubeConfigSecret(ctx, host, secret.Name, secret.Namespace, cfg); err != nil {
		t.Fatal(err)
	}
	if created != 0 || patched != 1 {
		t.Fatal("repair attempted to create or patch a missing Secret")
	}
}
