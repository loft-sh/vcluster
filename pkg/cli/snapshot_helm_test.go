package cli

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/find"
	"gotest.tools/v3/assert"
)

func TestValidateVClusterIsRunning(t *testing.T) {
	testTable := []struct {
		name        string
		status      find.Status
		expectedErr string
	}{
		{
			name:   "running",
			status: find.StatusRunning,
		},
		{
			name:        "paused",
			status:      find.StatusPaused,
			expectedErr: `cannot snapshot vCluster "my-vcluster" because it is not running (current status: "Paused")`,
		},
		{
			name:        "unknown",
			status:      find.StatusUnknown,
			expectedErr: `cannot snapshot vCluster "my-vcluster" because it is not running (current status: "Unknown")`,
		},
		{
			name:        "empty status is treated as unknown",
			status:      "",
			expectedErr: `cannot snapshot vCluster "my-vcluster" because it is not running (current status: "Unknown")`,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			vCluster := &find.VCluster{
				Name:   "my-vcluster",
				Status: testCase.status,
			}

			err := validateVClusterIsRunning(vCluster)
			if testCase.expectedErr == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, testCase.expectedErr)
			}
		})
	}
}
