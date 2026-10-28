package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/config"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestDescribeOutputString(t *testing.T) {
	tests := []struct {
		name string
		do   DescribeOutput
		want string
	}{
		{
			name: "empty user supplied vcluster.yaml",
			do: DescribeOutput{
				Name:         "test",
				Namespace:    "vcluster-test",
				Version:      "0.29.0",
				BackingStore: string(config.StoreTypeEmbeddedDatabase),
				Status:       "Running",
				Created:      metav1.NewTime(time.Unix(1759769661, 0).In(time.UTC)),
				Images: map[string]string{
					"apiServer": "ghcr.io/loft-sh/kubernetes:v1.33.4",
					"syncer":    "ghcr.io/loft-sh/vcluster-pro:0.29.0",
				},
			},
			want: `Name:           test
Namespace:      vcluster-test
Version:        0.29.0
Backing Store:  embedded-database
Created:        Mon, 06 Oct 2025 16:54:21 +0000
Status:         Running
Images:
  apiServer:  ghcr.io/loft-sh/kubernetes:v1.33.4
  syncer:     ghcr.io/loft-sh/vcluster-pro:0.29.0
`},
		{
			name: "user supplied vcluster.yaml",
			do: DescribeOutput{
				Name:         "test",
				Namespace:    "vcluster-test",
				Version:      "0.29.0",
				BackingStore: string(config.StoreTypeEmbeddedDatabase),
				Status:       "Running",
				Created:      metav1.NewTime(time.Unix(1759769661, 0).In(time.UTC)),
				Images: map[string]string{
					"apiServer": "ghcr.io/loft-sh/kubernetes:v1.33.4",
					"syncer":    "ghcr.io/loft-sh/vcluster-pro:0.29.0",
				},
				UserConfigYaml: ptr.To(`sync:
  toHost:
    serviceAccounts:
      enabled: true
`),
			},
			want: `Name:           test
Namespace:      vcluster-test
Version:        0.29.0
Backing Store:  embedded-database
Created:        Mon, 06 Oct 2025 16:54:21 +0000
Status:         Running
Images:
  apiServer:  ghcr.io/loft-sh/kubernetes:v1.33.4
  syncer:     ghcr.io/loft-sh/vcluster-pro:0.29.0

------------------- vcluster.yaml -------------------
sync:
  toHost:
    serviceAccounts:
      enabled: true
-----------------------------------------------------
Use --config-only to retrieve just the vcluster.yaml
`,
		},
		{
			name: "truncated user supplied vcluster.yaml",
			do: DescribeOutput{
				Name:         "test",
				Namespace:    "vcluster-test",
				Version:      "0.29.0",
				BackingStore: string(config.StoreTypeEmbeddedDatabase),
				Status:       "Running",
				Created:      metav1.NewTime(time.Unix(1759769661, 0).In(time.UTC)),
				Images: map[string]string{
					"apiServer": "ghcr.io/loft-sh/kubernetes:v1.33.4",
					"syncer":    "ghcr.io/loft-sh/vcluster-pro:0.29.0",
				},
				UserConfigYaml: ptr.To(strings.Repeat("line\n", 51)),
			},
			want: fmt.Sprintf(`Name:           test
Namespace:      vcluster-test
Version:        0.29.0
Backing Store:  embedded-database
Created:        Mon, 06 Oct 2025 16:54:21 +0000
Status:         Running
Images:
  apiServer:  ghcr.io/loft-sh/kubernetes:v1.33.4
  syncer:     ghcr.io/loft-sh/vcluster-pro:0.29.0

------------------- vcluster.yaml -------------------
%s... (truncated)
-----------------------------------------------------
Use --config-only to retrieve the full vcluster.yaml only
`, strings.Repeat("line\n", 50)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.do.String(), tt.want)
		})
	}
}

func TestConfigPartialUnmarshal(t *testing.T) {
	type args struct {
		configBytes []byte
	}
	tests := []struct {
		name                 string
		args                 args
		want                 *config.Config
		wantErr              bool
		wantBackingStoreType config.StoreType
	}{
		{
			name: "empty content",
			args: args{
				configBytes: []byte(""),
			},
			want:                 &config.Config{},
			wantErr:              false,
			wantBackingStoreType: config.StoreTypeEmbeddedDatabase,
		},
		{
			name: "parse only the controlPlane section",
			args: args{[]byte(`
controlPlane:
  advanced:
    defaultImageRegistry: ghcr.io
  distro:
    k8s:
      apiServer:
        enabled: true
      controllerManager:
        enabled: true
      image:
        registry: ghcr.io
        repository: loft-sh/kubernetes
        tag: v1.33.4
  statefulSet:
    image:
      registry: ghcr.io
      repository: loft-sh/vcluster-pro
telemetry:
  enabled: true

fieldNotMatchingConfigSchema:
  this: ["should", "be", "ignored"]
`)},
			want: &config.Config{
				ControlPlane: config.ControlPlane{
					Advanced: config.ControlPlaneAdvanced{
						DefaultImageRegistry: "ghcr.io",
					},
					Distro: config.Distro{
						K8S: config.DistroK8s{
							APIServer: config.DistroContainerEnabled{
								Enabled: true,
							},
							ControllerManager: config.DistroContainerEnabled{
								Enabled: true,
							},
							DistroCommon: config.DistroCommon{
								Image: config.Image{
									Registry:   "ghcr.io",
									Repository: "loft-sh/kubernetes",
									Tag:        "v1.33.4",
								},
							},
						},
					},
					StatefulSet: config.ControlPlaneStatefulSet{
						Image: config.Image{
							Registry:   "ghcr.io",
							Repository: "loft-sh/vcluster-pro",
						},
					},
				},

				// Telemetry section is expected to not be parsed.
				Telemetry: config.Telemetry{
					Enabled: false,
				},
			},
			wantErr:              false,
			wantBackingStoreType: config.StoreTypeEmbeddedDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configPartialUnmarshal(tt.args.configBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("configPartialUnmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.DeepEqual(t, got, tt.want)

			if gotBackingStoreType := got.BackingStoreType(); gotBackingStoreType != tt.wantBackingStoreType {
				t.Errorf("backingStoreType got = %v, want %v", gotBackingStoreType, tt.wantBackingStoreType)
			}
		})
	}
}

func TestGetImagesFromConfig(t *testing.T) {
	type args struct {
		c       *config.Config
		version string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "empty config",
			args: args{&config.Config{}, "0.29.0"},
			want: map[string]string{
				"syncer": "ghcr.io/loft-sh/library/vcluster-pro:0.29.0",
			},
		},
		{
			name: "syncer from config",
			args: args{&config.Config{
				ControlPlane: config.ControlPlane{
					Distro: config.Distro{
						K8S: config.DistroK8s{
							DistroCommon: config.DistroCommon{
								Image: config.Image{
									Registry:   "ghcr.io",
									Repository: "loft-sh/kubernetes",
									Tag:        "v1.33.4",
								},
							},
						},
					},
					StatefulSet: config.ControlPlaneStatefulSet{
						Image: config.Image{
							Registry:   "ghcr.io",
							Repository: "loft-sh/vcluster-pro",
							Tag:        "0.29.0",
						},
					},
				},
			}, "0.30.0"},
			want: map[string]string{
				"apiServer": "ghcr.io/loft-sh/kubernetes:v1.33.4",
				"syncer":    "ghcr.io/loft-sh/vcluster-pro:0.29.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.DeepEqual(t, getImagesFromConfig(tt.args.c, tt.args.version), tt.want)
		})
	}
}
