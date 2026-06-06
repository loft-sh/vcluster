package cli

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/find"
	"gotest.tools/v3/assert"
)

func TestValidateVClusterIsRunning(t *testing.T) {
	tests := []struct {
		name   string
		status find.Status
	}{
		{
			name:   "running",
			status: find.StatusRunning,
		},
		{
			name:   "paused",
			status: find.StatusPaused,
		},
		{
			name:   "workload sleeping",
			status: find.StatusWorkloadSleeping,
		},
		{
			name:   "scaled down",
			status: find.StatusScaledDown,
		},
		{
			name:   "unknown",
			status: find.StatusUnknown,
		},
		{
			name:   "pod status pending",
			status: find.Status("Pending"),
		},
		{
			name:   "pod status crash looping",
			status: find.Status("CrashLoopBackOff"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVClusterIsRunning(&find.VCluster{
				Name:   "my-vcluster",
				Status: tt.status,
			})
			if tt.status == find.StatusRunning {
				assert.NilError(t, err)
				return
			}
			assert.ErrorContains(t, err, "my-vcluster")
			assert.ErrorContains(t, err, string(tt.status))
		})
	}
}
