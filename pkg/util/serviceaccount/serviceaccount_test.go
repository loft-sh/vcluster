package serviceaccount

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/log"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestTokenRequestPreservesIssuerExpiry(t *testing.T) {
	for _, lifetime := range []int64{0, 3600} {
		api := fake.NewClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "scaling"}}, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "scaler", Namespace: "scaling"}})
		expiry := metav1.NewTime(time.Now().Add(10 * time.Minute))
		api.PrependReactor("create", "serviceaccounts", func(action clienttesting.Action) (bool, runtime.Object, error) {
			if action.GetSubresource() != "token" {
				return false, nil, nil
			}
			request := action.(clienttesting.CreateAction).GetObject().(*authenticationv1.TokenRequest)
			want := lifetime
			if want == 0 {
				want = 10 * 365 * 24 * 60 * 60
			}
			if *request.Spec.ExpirationSeconds != want {
				t.Fatalf("requested lifetime: got %d, want %d", *request.Spec.ExpirationSeconds, want)
			}
			return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: "test-token", ExpirationTimestamp: expiry}}, nil
		})
		status, err := CreateServiceAccountToken(context.Background(), api, "scaler", "scaling", "", lifetime, log.New("test"))
		if err != nil {
			t.Fatal(err)
		}
		if status.Token != "test-token" || !status.ExpirationTimestamp.Equal(&expiry) {
			t.Fatal("lost issuer token or expiry")
		}
		for _, action := range api.Actions() {
			if action.GetResource().Resource == "clusterrolebindings" {
				t.Fatal("preprovisioned exports must not modify tenant RBAC")
			}
		}
	}
}
