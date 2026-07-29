package legacyconfig

import (
	"testing"
)

func TestLegacyK8s_UnmarshalYAMLStrict(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid: k8s",
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
sync:
  nodes:
   enabled: true
telemetry:
  disabled: false
`),
			},
			wantErr: false,
		},
		{
			name: "Valid: eks",
			args: args{
				data: []byte(`
api:
  image: public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.28.2-eks-1-28-6
controller:
  image: public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.28.2-eks-1-28-6
coredns:
  image: public.ecr.aws/eks-distro/coredns/coredns:v1.10.1-eks-1-28-6
etcd:
  image: public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.9-eks-1-28-6
service:
  type: NodePort
sync:
  nodes:
    enabled: true
telemetry:
  disabled: false
`),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &LegacyK8s{}
			if err := c.UnmarshalYAMLStrict(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("LegacyK8s.UnmarshalYAMLStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
