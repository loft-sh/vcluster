package config

import (
	"bytes"
	_ "embed"
	"io"
	"testing"
)

func TestConfig_DecodeYAML(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid: yaml",
			args: args{
				r: bytes.NewReader([]byte(`
foo:
  bar: baz
`)),
			},
			wantErr: true,
		},
		{
			name: "Invalid: json",
			args: args{
				r: bytes.NewReader([]byte(`
{
  "foo": {
    "bar": "baz"
  }
}
`)),
			},
			wantErr: true,
		},
		{
			name: "Invalid: Old values format",
			args: args{
				r: bytes.NewReader([]byte(`
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
`)),
			},
			wantErr: true,
		},
		{
			name: "Success: New values format",
			args: args{
				r: bytes.NewReader([]byte(`
controlPlane:
  distro:
    k8s:
      enabled: true
`)),
			},
			wantErr: false,
		},
		{
			name: "Success: New values format (json)",
			args: args{
				r: bytes.NewReader([]byte(`
{
  "controlPlane": {
    "distro": {
      "k8s": {
        "enabled": true
      }
    }
  }
}
`)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			if err := c.DecodeYAML(tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("Config.DecodeYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
