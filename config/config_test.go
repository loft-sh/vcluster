package config

import (
	_ "embed"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"sigs.k8s.io/yaml"
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
		config   *Config
		name     string
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
			name: "Custom Resource Syncing to host is configured",
			config: &Config{
				Sync: Sync{
					ToHost: SyncToHost{
						CustomResources: map[string]SyncToHostCustomResource{
							"demo": {},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Custom Resource Syncing from host is configured",
			config: &Config{
				Sync: Sync{
					FromHost: SyncFromHost{
						CustomResources: map[string]SyncFromHostCustomResource{
							"demo": {},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Hybrid scheduling is enabled",
			config: &Config{
				Sync: Sync{
					ToHost: SyncToHost{
						Pods: SyncPods{
							HybridScheduling: HybridScheduling{
								Enabled: true,
							},
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

// We changed sync.toHost.pods.rewriteHosts.initContainer.image from a string to an object in 0.27.0.
// We parse the previously used config on upgrade, so it must be backwards compatible.
func TestImage_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected Image
	}{
		{
			name: "image as object",
			yaml: `registry: registry:5000
repository: some/repo
tag: sometag`,
			expected: Image{
				Registry:   "registry:5000",
				Repository: "some/repo",
				Tag:        "sometag",
			},
		},
		{
			name: "image as string",
			yaml: "registry:5000/some/repo:sometag",
			expected: Image{
				Registry:   "registry:5000",
				Repository: "some/repo",
				Tag:        "sometag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual Image
			err := yaml.Unmarshal([]byte(tt.yaml), &actual)
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, tt.expected)
		})
	}
}

func TestImage_String(t *testing.T) {
	testCases := []struct {
		name     string
		image    Image
		expected string
	}{
		{
			name: "complete image reference",
			image: Image{
				Registry:   "registry.k8s.io",
				Repository: "coredns/coredns",
				Tag:        "1.11.3",
			},
			expected: "registry.k8s.io/coredns/coredns:1.11.3",
		},
		{
			name: "may omit registry",
			image: Image{
				Repository: "coredns/coredns",
				Tag:        "1.11.3",
			},
			expected: "coredns/coredns:1.11.3",
		},
		{
			name: "may omit registry and repo",
			image: Image{
				Repository: "alpine",
				Tag:        "3.20",
			},
			expected: "alpine:3.20",
		},
		{
			name: "may omit tag",
			image: Image{
				Repository: "alpine",
			},
			expected: "alpine",
		},
		{
			name: "omit repo but not registry is library",
			image: Image{
				Registry:   "ghcr.io",
				Repository: "alpine",
				Tag:        "3.20",
			},
			expected: "ghcr.io/library/alpine:3.20",
		},
		{
			name: "registry may have port",
			image: Image{
				Registry:   "host.docker.internal:5000",
				Repository: "coredns/coredns",
				Tag:        "1.11.3",
			},
			expected: "host.docker.internal:5000/coredns/coredns:1.11.3",
		},
		{
			name: "registry with port and omit tag",
			image: Image{
				Registry:   "localhost:5000",
				Repository: "coredns/coredns",
			},
			expected: "localhost:5000/coredns/coredns",
		},
		{
			name:     "empty image is nil value",
			image:    Image{},
			expected: "",
		},
	}

	for _, tt := range testCases {
		t.Run("String(): "+tt.name, func(t *testing.T) {
			if actual := tt.image.String(); actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})

		t.Run("ParseImageRef(): "+tt.name, func(t *testing.T) {
			var image Image
			ParseImageRef(tt.expected, &image)
			assert.Check(t, cmp.DeepEqual(tt.image, image))
		})
	}
}
