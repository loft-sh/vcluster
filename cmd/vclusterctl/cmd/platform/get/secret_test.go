package get

import (
	"errors"
	"testing"

	managementv1 "github.com/loft-sh/api/v5/pkg/apis/management/v1"
	loftclient "github.com/loft-sh/api/v5/pkg/clientset/versioned"
	loftfake "github.com/loft-sh/api/v5/pkg/clientset/versioned/fake"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"
)

type secretTestClient struct {
	kube.Interface
	loftClient loftclient.Interface
}

func (c *secretTestClient) Loft() loftclient.Interface {
	return c.loftClient
}

func TestListSharedSecretsWithFallback(t *testing.T) {
	listErr := errors.New("list failed")
	tests := []struct {
		name            string
		namespace       string
		secretNamespace string
		failedNamespace string
		wantError       bool
	}{
		{name: "all namespaces empty"},
		{name: "explicit namespace and fallbacks empty", namespace: "custom"},
		{name: "default namespace has a secret", secretNamespace: clihelper.DefaultPlatformNamespace},
		{name: "legacy namespace has a secret", secretNamespace: clihelper.LegacyPlatformNamespace},
		{name: "explicit namespace has a secret", namespace: "custom", secretNamespace: "custom"},
		{name: "list errors are preserved", failedNamespace: clihelper.DefaultPlatformNamespace, wantError: true},
		{name: "fallback succeeds after an error", failedNamespace: clihelper.DefaultPlatformNamespace, secretNamespace: clihelper.LegacyPlatformNamespace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []runtime.Object
			if tt.secretNamespace != "" {
				objects = append(objects, &managementv1.SharedSecret{
					ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: tt.secretNamespace},
				})
			}
			client := loftfake.NewSimpleClientset(objects...)
			client.PrependReactor("list", "sharedsecrets", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if action.GetNamespace() == tt.failedNamespace {
					return true, nil, listErr
				}
				return false, nil, nil
			})
			cmd := &SecretCmd{log: log.GetInstance()}
			secrets, err := cmd.listSharedSecretsWithFallback(t.Context(), tt.namespace, &secretTestClient{loftClient: client})
			if tt.wantError {
				if !errors.Is(err, listErr) {
					t.Fatalf("expected list error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if secrets == nil {
				t.Fatal("successful list must return a non-nil list, even when no secrets exist")
			}
			if tt.secretNamespace == "" {
				if len(secrets.Items) != 0 {
					t.Fatalf("expected no secrets, got %v", secrets.Items)
				}
			} else if len(secrets.Items) != 1 || secrets.Items[0].Namespace != tt.secretNamespace || secrets.Items[0].Name != "test-secret" {
				t.Fatalf("expected test-secret in namespace %s, got %v", tt.secretNamespace, secrets.Items)
			}
		})
	}
}
