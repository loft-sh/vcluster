package setup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

func TestExportRenewalUsesGrantedLifetimeAndRetriesFailures(t *testing.T) {
	now := time.Now()
	calls := 0
	fail := false
	r := &exportRenewal{
		now: func() time.Time { return now }, rotating: true,
		write: func(context.Context) (time.Time, error) {
			calls++
			if fail {
				return time.Time{}, errors.New("temporary write failure")
			}
			// The issuer grants ten minutes, even if an hour was requested.
			return now.Add(10 * time.Minute), nil
		},
	}
	ctx := context.Background()
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if r.next.Before(now.Add(360*time.Second)) || r.next.After(now.Add(400*time.Second)) {
		t.Fatalf("renewal not based on actual granted lifetime: %s", r.next.Sub(now))
	}
	if err := r.reconcile(ctx); err != nil || calls != 1 {
		t.Fatalf("unexpected early renewal: %v, calls=%d", err, calls)
	}
	now = r.next
	fail = true
	if err := r.reconcile(ctx); err == nil {
		t.Fatal("expected failed write")
	}
	if r.next.After(now) {
		t.Fatal("failed write postponed retry")
	}
	fail = false
	if err := r.reconcile(ctx); err != nil || calls != 3 {
		t.Fatalf("renewal did not retry: %v, calls=%d", err, calls)
	}
}

func TestExportRenewalHonorsConfiguredPercentage(t *testing.T) {
	for _, percentage := range []int32{1, 33, 50, 80, 99} {
		now := time.Now()
		r := &exportRenewal{
			now: func() time.Time { return now }, rotating: true,
			renewBeforePercentage: ptr.To(percentage),
			write: func(context.Context) (time.Time, error) {
				// Scheduling must use the granted lifetime, even when the request was longer.
				return now.Add(10 * time.Minute), nil
			},
		}
		if err := r.reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
		delay := 6 * time.Second * time.Duration(100-percentage)
		if r.next.After(now.Add(delay)) || r.next.Before(now.Add(delay*9/10)) {
			t.Fatalf("renewal with %d%% remaining scheduled at %s, expected %s to %s", percentage, r.next.Sub(now), delay*9/10, delay)
		}
	}
}

func TestExportRenewalPreservesWindowAcrossSlowWrites(t *testing.T) {
	issuedAt := time.Now()
	now := issuedAt
	r := &exportRenewal{
		now: func() time.Time { return now }, rotating: true,
		renewBeforePercentage: ptr.To[int32](50),
		write: func(context.Context) (time.Time, error) {
			now = now.Add(time.Minute)
			return issuedAt.Add(10 * time.Minute), nil
		},
	}
	if err := r.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.next.After(issuedAt.Add(5 * time.Minute)) {
		t.Fatal("slow export write shortened the configured renewal window")
	}
}

func TestLegacyExportIsWrittenOnce(t *testing.T) {
	calls := 0
	r := &exportRenewal{now: time.Now, write: func(context.Context) (time.Time, error) { calls++; return time.Time{}, nil }}
	for range 3 {
		if err := r.reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("legacy export issued %d credentials", calls)
	}
}

func TestEmptyRotationRetainsLegacyTokenLifetime(t *testing.T) {
	for _, rotation := range []*vclusterconfig.ExportKubeConfigTokenRotation{nil, {}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var response any
			switch r.Method + " " + r.URL.Path {
			case "POST /api/v1/namespaces/default/serviceaccounts/scaler/token":
				request := &authenticationv1.TokenRequest{}
				if err := json.NewDecoder(r.Body).Decode(request); err != nil {
					t.Error(err)
					http.Error(w, "invalid token request", http.StatusBadRequest)
					return
				}
				if request.Spec.ExpirationSeconds == nil || *request.Spec.ExpirationSeconds != 315360000 {
					t.Errorf("empty rotation changed the legacy token lifetime: %v", request.Spec.ExpirationSeconds)
				}
				response = &authenticationv1.TokenRequest{
					TypeMeta: metav1.TypeMeta{APIVersion: "authentication.k8s.io/v1", Kind: "TokenRequest"},
					Status: authenticationv1.TokenRequestStatus{
						Token: "legacy-token", ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
					},
				}
			case "GET /api/v1/namespaces/default/serviceaccounts/scaler":
				response = &corev1.ServiceAccount{
					TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
					ObjectMeta: metav1.ObjectMeta{Name: "scaler", Namespace: "default"},
				}
			case "GET /api/v1/namespaces/default":
				response = &corev1.Namespace{
					TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				}
			default:
				t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
				return
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Error(err)
			}
		}))
		raw := &clientcmdapi.Config{
			CurrentContext: "source",
			Contexts:       map[string]*clientcmdapi.Context{"source": {Cluster: "source", AuthInfo: "source"}},
			Clusters:       map[string]*clientcmdapi.Cluster{"source": {Server: "https://virtual.example.com"}},
			AuthInfos:      map[string]*clientcmdapi.AuthInfo{"source": {}},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		virtual := &rest.Config{Host: server.URL, ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"}}
		cfg, _, err := CreateVClusterKubeConfigForExport(ctx, virtual, raw, CreateKubeConfigOptions{
			ExportKubeConfig: vclusterconfig.ExportKubeConfigProperties{
				ServiceAccount: vclusterconfig.ExportKubeConfigServiceAccount{Name: "scaler", Namespace: "default", TokenRotation: rotation},
			},
		})
		cancel()
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.AuthInfos["source"].Token != "legacy-token" {
			t.Fatal("empty rotation did not export the legacy credential")
		}
	}
}

func TestExportEventsRepairWithoutIssuingTokensOrPostponingRenewal(t *testing.T) {
	now := time.Now()
	issued, repaired := 0, 0
	failRepair := false
	r := &exportRenewal{
		now: func() time.Time { return now }, rotating: true,
		write: func(context.Context) (time.Time, error) {
			issued++
			return now.Add(10 * time.Minute), nil
		},
		repair: func(context.Context) error {
			repaired++
			if failRepair {
				return errors.New("temporary patch failure")
			}
			return nil
		},
	}
	ctx := context.Background()
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := r.next
	for range 10 {
		if err := r.reconcile(ctx); err != nil {
			t.Fatal(err)
		}
	}
	failRepair = true
	if err := r.reconcile(ctx); err == nil {
		t.Fatal("expected repair error")
	}
	if issued != 1 || repaired != 11 || r.next != deadline {
		t.Fatal("Secret events or a repair failure changed issuance or the renewal deadline")
	}
	now = deadline
	if err := r.reconcile(ctx); err != nil || issued != 2 {
		t.Fatalf("watch events prevented scheduled renewal: %v", err)
	}
}

func TestExportRejectsInvalidLifetimeBeforeIssuance(t *testing.T) {
	for _, sa := range []vclusterconfig.ExportKubeConfigServiceAccount{
		{Name: "scaler", TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{RenewBeforePercentage: ptr.To[int32](33)}},
		{Name: "scaler", TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{Expiration: "-1s"}},
		{Name: "scaler", TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{Expiration: "599s"}},
		{Name: "scaler", TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{Expiration: "4294967297s"}},
		{TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{Expiration: "10m"}},
	} {
		_, _, err := CreateVClusterKubeConfigForExport(context.Background(), nil, nil, CreateKubeConfigOptions{
			ExportKubeConfig: vclusterconfig.ExportKubeConfigProperties{ServiceAccount: sa},
		})
		if err == nil {
			t.Fatalf("accepted invalid lifetime: %#v", sa)
		}
	}
}

func TestRotatingExportsRejectUnsupportedDestinationsBeforeStarting(t *testing.T) {
	for _, standalone := range []bool{true, false} {
		options := &config.VirtualClusterConfig{}
		options.ExportKubeConfig.ServiceAccount = vclusterconfig.ExportKubeConfigServiceAccount{Name: "scaler", TokenRotation: &vclusterconfig.ExportKubeConfigTokenRotation{Expiration: "10m"}}
		options.ControlPlane.Standalone.Enabled = standalone
		if !standalone {
			// An empty additional destination resolves to the default Secret.
			options.ExportKubeConfig.AdditionalSecrets = []vclusterconfig.ExportKubeConfigAdditionalSecretReference{{}}
		}
		if err := startKubeConfigExports(context.Background(), nil, "host", nil, options, nil); err == nil {
			t.Fatalf("accepted unsupported destination, standalone=%v", standalone)
		}
	}
}
