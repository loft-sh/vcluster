package toleration

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestParseToleration(t *testing.T) {
	type args struct {
		st string
	}
	tests := []struct {
		name    string
		args    args
		want    corev1.Toleration
		wantErr bool
	}{
		{
			name: "Should get toleration with Operator exists",
			args: args{
				st: "*",
			},
			want: corev1.Toleration{
				Operator: corev1.TolerationOpExists,
			},
			wantErr: false,
		},
		{
			name: "Should get toleration with key",
			args: args{
				st: "key",
			},
			want: corev1.Toleration{
				Key: "key",
			},
			wantErr: false,
		},
		{
			name: "Should get toleration with key and value",
			args: args{
				st: "key=value",
			},
			want: corev1.Toleration{
				Key:   "key",
				Value: "value",
			},
			wantErr: false,
		},
		{
			name: "Should get toleration with key value and effect",
			args: args{
				st: "key=value:NoSchedule",
			},
			want: corev1.Toleration{
				Key:      "key",
				Value:    "value",
				Effect:   corev1.TaintEffectNoSchedule,
				Operator: corev1.TolerationOpEqual,
			},
			wantErr: false,
		},
		{
			name: "Should get toleration with key and effect",
			args: args{
				st: "key:NoSchedule",
			},
			want: corev1.Toleration{
				Key:      "key",
				Effect:   corev1.TaintEffectNoSchedule,
				Operator: corev1.TolerationOpExists,
			},
			wantErr: false,
		},
		{
			name: "Should get bad label error",
			args: args{
				st: "key=value,wrong:NoSchedule",
			},
			want:    corev1.Toleration{},
			wantErr: true,
		},
		{
			name: "Should get invalid toleration",
			args: args{
				st: "key=value:Effec:Effect",
			},
			want:    corev1.Toleration{},
			wantErr: true,
		},
		{
			name: "Should get invalid toleration",
			args: args{
				st: "key=value=value",
			},
			want:    corev1.Toleration{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToleration(tt.args.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToleration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseToleration() = %v, want %v", got, tt.want)
			}
		})
	}
}
