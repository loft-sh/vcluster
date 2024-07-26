package config

import (
	_ "embed"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestConfig_Diff(t *testing.T) {
	tests := []struct {
		name string

		config   func(c *Config)
		expected string
	}{
		{
			name: "Simple",
			config: func(c *Config) {
				c.Sync.ToHost.Services.Enabled = false
			},
			expected: `sync:
  toHost:
    services:
      enabled: false`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultConfig, err := NewDefaultConfig()
			assert.NilError(t, err)

			toConfig, err := NewDefaultConfig()
			assert.NilError(t, err)

			tt.config(toConfig)

			expectedConfig, err := Diff(defaultConfig, toConfig)
			assert.NilError(t, err)
			assert.Equal(t, tt.expected, strings.TrimSpace(expectedConfig))
		})
	}
}

func TestConfig_UnmarshalYAMLStrict(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid: yaml",
			args: args{
				data: []byte(`
foo:
  bar: baz
`),
			},
			wantErr: true,
		},
		{
			name: "Invalid: json",
			args: args{
				data: []byte(`
{
  "foo": {
    "bar": "baz"
  }
}
`),
			},
			wantErr: true,
		},
		{
			name: "Invalid: Old values format",
			args: args{
				data: []byte(`
api:
  image: registry.k8s.io/kube-apiserver:v1.29.0
controller:
  image: registry.k8s.io/kube-controller-manager:v1.29.0
etcd:
  image: registry.k8s.io/etcd:3.5.10-0
scheduler:
  image: registry.k8s.io/kube-scheduler:v1.29.0
service:
  type: NodePort
serviceCIDR: 10.96.0.0/16
sync:
  nodes:
   enabled: true
telemetry:
  disabled: false
`),
			},
			wantErr: true,
		},
		{
			name: "Success: New values format",
			args: args{
				data: []byte(`
controlPlane:
  distro:
    k8s:
      enabled: true
`),
			},
			wantErr: false,
		},
		{
			name: "Success: New values format (json)",
			args: args{
				data: []byte(`
{
  "controlPlane": {
    "distro": {
      "k8s": {
        "enabled": true
      }
    }
  }
}
`),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			if err := c.UnmarshalYAMLStrict(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Config.UnmarshalYAMLStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsProFeatureEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "No pro features",
			config:   &Config{},
			expected: false,
		},
		{
			name: "Empty ResolveDNS",
			config: &Config{
				Networking: Networking{
					ResolveDNS: []ResolveDNS{},
				},
			},
			expected: false,
		},
		{
			name: "ResolveDNS used",
			config: &Config{
				Networking: Networking{
					ResolveDNS: []ResolveDNS{
						{
							Hostname: "wikipedia.com",
							Target: ResolveDNSTarget{
								Hostname: "en.wikipedia.org",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Central Admission Control validating webhooks used",
			config: &Config{
				Policies: Policies{
					CentralAdmission: CentralAdmission{
						ValidatingWebhooks: []ValidatingWebhookConfiguration{
							{
								Kind: "ValidatingWebhookConfiguration",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Central Admission Control mutating webhooks used",
			config: &Config{
				Policies: Policies{
					CentralAdmission: CentralAdmission{
						MutatingWebhooks: []MutatingWebhookConfiguration{
							{
								Kind: "MutatingWebhookConfiguration",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Embedded etcd not used",
			config: &Config{
				ControlPlane: ControlPlane{
					BackingStore: BackingStore{
						Etcd: Etcd{
							Embedded: EtcdEmbedded{
								Enabled: false,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Embedded etcd used",
			config: &Config{
				ControlPlane: ControlPlane{
					BackingStore: BackingStore{
						Etcd: Etcd{
							Embedded: EtcdEmbedded{
								Enabled: true,
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Host Path Mapper not used",
			config: &Config{
				ControlPlane: ControlPlane{
					HostPathMapper: HostPathMapper{
						Enabled: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "Host Path Mapper used",
			config: &Config{
				ControlPlane: ControlPlane{
					HostPathMapper: HostPathMapper{
						Enabled: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "Central Host Path Mapper not used",
			config: &Config{
				ControlPlane: ControlPlane{
					HostPathMapper: HostPathMapper{
						Central: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "Central Host Path Mapper used",
			config: &Config{
				ControlPlane: ControlPlane{
					HostPathMapper: HostPathMapper{
						Central: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "Pro Sync Settings not used",
			config: &Config{
				Experimental: Experimental{
					SyncSettings: ExperimentalSyncSettings{
						DisableSync:              false,
						RewriteKubernetesService: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "Pro Sync Setting disableSync used",
			config: &Config{
				Experimental: Experimental{
					SyncSettings: ExperimentalSyncSettings{
						DisableSync: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "Pro Sync Setting rewriteKubernetesService used",
			config: &Config{
				Experimental: Experimental{
					SyncSettings: ExperimentalSyncSettings{
						RewriteKubernetesService: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "Isolated Control Plane not used",
			config: &Config{
				Experimental: Experimental{
					IsolatedControlPlane: ExperimentalIsolatedControlPlane{
						Enabled: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "Isolated Control Plane used",
			config: &Config{
				Experimental: Experimental{
					IsolatedControlPlane: ExperimentalIsolatedControlPlane{
						Enabled: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "Deny Proxy Requests not used",
			config: &Config{
				Experimental: Experimental{
					DenyProxyRequests: []DenyRule{},
				},
			},
			expected: false,
		},
		{
			name: "Deny Proxy Requests used",
			config: &Config{
				Experimental: Experimental{
					DenyProxyRequests: []DenyRule{
						{
							Name: "test",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "External Platform configuration used",
			config: &Config{
				External: map[string]ExternalConfig{
					"platform": map[string]interface{}{
						"autoSleep": map[string]interface{}{
							"afterInactivity": 300,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.config.IsProFeatureEnabled(), tt.expected)
		})
	}
}
