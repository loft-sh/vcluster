package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestDeleteMultiNamespaceVClusterWorkloadsReturnsPodListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/namespaces" {
			_ = json.NewEncoder(w).Encode(&corev1.NamespaceList{
				TypeMeta: metav1.TypeMeta{Kind: "NamespaceList", APIVersion: "v1"},
				Items:    []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "host-namespace"}}},
			})
			return
		}

		// listing the pods of the namespace fails
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(&metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure,
			Message:  "cannot list pods",
			Reason:   metav1.StatusReasonInternalError,
			Code:     http.StatusInternalServerError,
		})
	}))
	defer server.Close()

	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	err = DeleteMultiNamespaceVClusterWorkloads(context.Background(), client, "my-vcluster", "my-namespace", log.Discard)
	if err == nil {
		t.Fatal("expected an error because the pods could not be listed, got nil")
	}
}
