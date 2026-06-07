package cli

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/find"
	"gotest.tools/v3/assert"
)

func TestValidateVClusterIsRunning(t *testing.T) {
	tests := []struct {
		name      string
		status    find.Status
		wantErr   bool
		wantState find.Status // state named in the error message
	}{
		{
			name:   "running",
			status: find.StatusRunning,
		},
		{
			name:      "paused",
			status:    find.StatusPaused,
			wantErr:   true,
			wantState: find.StatusPaused,
		},
		{
			name:      "workload sleeping",
			status:    find.StatusWorkloadSleeping,
			wantErr:   true,
			wantState: find.StatusWorkloadSleeping,
		},
		{
			name:      "scaled down",
			status:    find.StatusScaledDown,
			wantErr:   true,
			wantState: find.StatusScaledDown,
		},
		{
			name:      "unknown",
			status:    find.StatusUnknown,
			wantErr:   true,
			wantState: find.StatusUnknown,
		},
		{
			name:      "pod status pending",
			status:    find.Status("Pending"),
			wantErr:   true,
			wantState: find.Status("Pending"),
		},
		{
			name:      "pod status crash looping",
			status:    find.Status("CrashLoopBackOff"),
			wantErr:   true,
			wantState: find.Status("CrashLoopBackOff"),
		},
		{
			name:      "empty status is reported as unknown",
			status:    find.Status(""),
			wantErr:   true,
			wantState: find.StatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVClusterIsRunning(&find.VCluster{
				Name:   "my-vcluster",
				Status: tt.status,
			})
			if !tt.wantErr {
				assert.NilError(t, err)
				return
			}
			assert.ErrorContains(t, err, "my-vcluster")
			assert.ErrorContains(t, err, string(tt.wantState))
		})
	}
}
