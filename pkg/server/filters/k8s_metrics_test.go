package filters

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rawconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	vtesting "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestMetricsRestConfig(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		embeddedEtcd bool
		externalEtcd bool
		deployedEtcd bool
		externalDB   bool
		embeddedDB   bool
		wantHost     string
		wantNil      bool
	}{
		{
			name:     "old controller manager route",
			path:     "/controller-manager/metrics",
			wantHost: controllerManagerMetricsHost,
		},
		{
			name:     "new controller manager route",
			path:     "/metrics/controller-manager",
			wantHost: controllerManagerMetricsHost,
		},
		{
			name:     "old scheduler route",
			path:     "/scheduler/metrics",
			wantHost: schedulerMetricsHost,
		},
		{
			name:     "new scheduler route",
			path:     "/metrics/scheduler",
			wantHost: schedulerMetricsHost,
		},
		{
			name:         "embedded etcd route",
			path:         "/metrics/etcd",
			embeddedEtcd: true,
			wantHost:     localBackingStoreMetricsURL,
		},
		{
			name:    "etcd route disabled without embedded etcd",
			path:    "/metrics/etcd",
			wantNil: true,
		},
		{
			name:     "kine route default backing store (embedded sqlite)",
			path:     "/metrics/kine",
			wantHost: localBackingStoreMetricsURL,
		},
		{
			name:       "kine route explicit embedded database",
			path:       "/metrics/kine",
			embeddedDB: true,
			wantHost:   localBackingStoreMetricsURL,
		},
		{
			name:       "kine route external database",
			path:       "/metrics/kine",
			externalDB: true,
			wantHost:   localBackingStoreMetricsURL,
		},
		{
			name:         "kine route disabled with embedded etcd",
			path:         "/metrics/kine",
			embeddedEtcd: true,
			wantNil:      true,
		},
		{
			name:         "kine route disabled with deployed etcd",
			path:         "/metrics/kine",
			deployedEtcd: true,
			wantNil:      true,
		},
		{
			name:         "kine route disabled with external etcd",
			path:         "/metrics/kine",
			externalEtcd: true,
			wantNil:      true,
		},
		{
			name:    "unrelated route",
			path:    "/metrics",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerCtx := &synccontext.RegisterContext{
				Config: &config.VirtualClusterConfig{
					Config: rawconfig.Config{
						ControlPlane: rawconfig.ControlPlane{
							BackingStore: rawconfig.BackingStore{
								Etcd: rawconfig.Etcd{
									Embedded: rawconfig.EtcdEmbedded{
										Enabled: tt.embeddedEtcd,
									},
									Deploy: rawconfig.EtcdDeploy{
										Enabled: tt.deployedEtcd,
									},
									External: rawconfig.EtcdExternal{
										Enabled: tt.externalEtcd,
									},
								},
								Database: rawconfig.Database{
									Embedded: rawconfig.DatabaseKine{
										Enabled: tt.embeddedDB,
									},
									External: rawconfig.ExternalDatabaseKine{
										DatabaseKine: rawconfig.DatabaseKine{
											Enabled: tt.externalDB,
										},
									},
								},
							},
						},
					},
				},
				VirtualManager: vtesting.NewFakeManager(nil),
			}

			got := metricsRestConfig(tt.path, registerCtx)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil config, got host %q", got.Host)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected config for %q", tt.path)
			}
			if got.Host != tt.wantHost {
				t.Fatalf("expected host %q, got %q", tt.wantHost, got.Host)
			}
		})
	}
}

func TestWithK8sMetricsSyncerRoute(t *testing.T) {
	probe := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vcluster_syncer_metrics_route_probe_total",
		Help: "Probe metric asserting /metrics/syncer serves the controller-runtime registry.",
	})
	ctrlmetrics.Registry.MustRegister(probe)
	defer ctrlmetrics.Registry.Unregister(probe)
	probe.Inc()

	registerCtx := &synccontext.RegisterContext{
		Config:         &config.VirtualClusterConfig{},
		VirtualManager: vtesting.NewFakeManager(nil),
	}

	nextCalled := false
	h := WithK8sMetrics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}), registerCtx)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics/syncer", nil))
	if nextCalled {
		t.Fatal("expected /metrics/syncer to be served in-process, but the request fell through")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "vcluster_syncer_metrics_route_probe_total 1") {
		t.Fatalf("expected body to contain the probe metric from the controller-runtime registry, got:\n%s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !nextCalled {
		t.Fatal("expected unrelated path to fall through to the next handler")
	}
}
