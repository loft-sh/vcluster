package cmd

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"gotest.tools/v3/assert"
)

func TestDeleteArgs(t *testing.T) {
	testCases := []struct {
		name          string
		flags         []string
		args          []string
		expectedError string
	}{
		{
			name: "single name",
			args: []string{"my-vcluster"},
		},
		{
			name:          "missing name",
			args:          []string{},
			expectedError: `please specify missing: "VCLUSTER_NAME"`,
		},
		{
			name:          "too many names",
			args:          []string{"a", "b"},
			expectedError: `extra arguments: "b"`,
		},
		{
			name:  "all without a name",
			flags: []string{"--all"},
			args:  []string{},
		},
		{
			name:  "shorthand without a name",
			flags: []string{"-A"},
			args:  []string{},
		},
		{
			name:          "all with a name",
			flags:         []string{"--all"},
			args:          []string{"my-vcluster"},
			expectedError: `cannot combine a virtual cluster name ("my-vcluster") with --all`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deleteCmd := NewDeleteCmd(&flags.GlobalFlags{})
			assert.NilError(t, deleteCmd.ParseFlags(testCase.flags))

			err := deleteCmd.ValidateArgs(testCase.args)
			if testCase.expectedError == "" {
				assert.NilError(t, err)
				return
			}

			assert.ErrorContains(t, err, testCase.expectedError)
		})
	}
}
