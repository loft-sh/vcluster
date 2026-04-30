package filters

import (
	"testing"

	rawconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	vtesting "github.com/loft-sh/vcluster/pkg/util/testing"
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
			wantHost:     embeddedEtcdMetricsHost,
		},
		{
			name:    "etcd route disabled without embedded etcd",
			path:    "/metrics/etcd",
			wantNil: true,
		},
		{
			name:     "kine route default backing store (embedded sqlite)",
			path:     "/metrics/kine",
			wantHost: kineMetricsHost,
		},
		{
			name:       "kine route explicit embedded database",
			path:       "/metrics/kine",
			embeddedDB: true,
			wantHost:   kineMetricsHost,
		},
		{
			name:       "kine route external database",
			path:       "/metrics/kine",
			externalDB: true,
			wantHost:   kineMetricsHost,
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
