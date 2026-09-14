package platform

import (
	"strconv"
	"testing"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCanAutoWakeup(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name        string
		annotations map[string]string
		wantBlocked bool
	}{
		{
			name:        "no force-duration → allow",
			annotations: nil,
			wantBlocked: false,
		},
		{
			name: "force-duration with sleeping-since within window → block",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "600",
				clusterv1.SleepModeSleepingSinceAnnotation: strconv.FormatInt(now-60, 10),
			},
			wantBlocked: true,
		},
		{
			name: "force-duration but window elapsed → allow",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "60",
				clusterv1.SleepModeSleepingSinceAnnotation: strconv.FormatInt(now-3600, 10),
			},
			wantBlocked: false,
		},
		{
			name: "force-duration=0 (sleep until explicit wake) → block",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "0",
				clusterv1.SleepModeSleepingSinceAnnotation: strconv.FormatInt(now-3600, 10),
			},
			wantBlocked: true,
		},
		{
			name: "force-duration without sleeping-since → allow",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "600",
			},
			wantBlocked: false,
		},
		{
			name: "malformed force-duration → allow (permissive)",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "not-a-number",
				clusterv1.SleepModeSleepingSinceAnnotation: strconv.FormatInt(now-60, 10),
			},
			wantBlocked: false,
		},
		{
			name: "negative force-duration → allow (permissive)",
			annotations: map[string]string{
				clusterv1.SleepModeForceDurationAnnotation: "-1",
				clusterv1.SleepModeSleepingSinceAnnotation: strconv.FormatInt(now-60, 10),
			},
			wantBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vci := &managementv1.VirtualClusterInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-tenant",
					Namespace:   "loft-p-my-project",
					Annotations: tt.annotations,
				},
			}
			err := canAutoWakeup(vci)
			if tt.wantBlocked && err == nil {
				t.Fatalf("expected block, got nil")
			}
			if !tt.wantBlocked && err != nil {
				t.Fatalf("expected allow, got error: %v", err)
			}
		})
	}
}
