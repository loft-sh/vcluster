package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/loft-sh/log"
	"gotest.tools/v3/assert"
)

func TestHelpRequested(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		expectedRequested bool
		expectedCmdPath   string
	}{
		{
			name:              "help swallowed by a value taking flag",
			args:              []string{"platform", "start", "--config", "-h"},
			expectedRequested: true,
			expectedCmdPath:   "vcluster platform start",
		},
		{
			name:              "regular help flag",
			args:              []string{"platform", "start", "--help"},
			expectedRequested: true,
			expectedCmdPath:   "vcluster platform start",
		},
		{
			name:              "root help",
			args:              []string{"-h"},
			expectedRequested: true,
			expectedCmdPath:   "vcluster",
		},
		{
			name:              "explicit value keeps the -h",
			args:              []string{"platform", "start", "--config=-h"},
			expectedRequested: false,
		},
		{
			name:              "after the terminator",
			args:              []string{"create", "my-vcluster", "--", "-h"},
			expectedRequested: false,
		},
		{
			name:              "shell completion is not hijacked",
			args:              []string{"__complete", "platform", "start", "--config", "-h"},
			expectedRequested: false,
		},
		{
			name:              "unknown command",
			args:              []string{"does-not-exist", "--config", "-h"},
			expectedRequested: false,
		},
		{
			name:              "no args",
			args:              []string{},
			expectedRequested: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rootCmd, _, err := BuildRoot(log.GetInstance())
			assert.NilError(t, err)

			requested, cmd := helpRequested(rootCmd, testCase.args)
			assert.Equal(t, requested, testCase.expectedRequested)

			if !testCase.expectedRequested {
				assert.Assert(t, cmd == nil, "expected no command, got %v", cmd)
				return
			}

			assert.Assert(t, cmd != nil, "expected a command for %v", testCase.args)
			assert.Equal(t, cmd.CommandPath(), testCase.expectedCmdPath)
		})
	}
}

// GUARD: fails when executeHelp stops rendering help the way cobra does, for
// example by printing it directly and skipping cobra's command and flag setup.
func TestExecuteHelpMatchesCobra(t *testing.T) {
	testCases := []struct {
		name string
		// args is what the user typed, with the help flag at risk of being swallowed.
		args []string
		// cobraArgs is the same command invoked so that cobra parses the help
		// flag itself. It must never contain a swallowed flag: cobra would then
		// run the command for real.
		cobraArgs []string
	}{
		{
			name:      "root",
			args:      []string{"-h"},
			cobraArgs: []string{"--help"},
		},
		{
			name:      "subcommand with a swallowed help flag",
			args:      []string{"platform", "start", "--config", "-h"},
			cobraArgs: []string{"platform", "start", "--help"},
		},
		{
			name:      "subcommand with a regular help flag",
			args:      []string{"platform", "start", "--help"},
			cobraArgs: []string{"platform", "start", "--help"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			want := captureCobraHelp(t, testCase.cobraArgs)
			assert.Assert(t, want != "", "cobra printed no help for %v", testCase.cobraArgs)

			got := captureExecuteHelp(t, testCase.args)
			assert.Equal(t, got, want,
				"executeHelp no longer matches cobra's own help output, so it skips setup cobra does. Route the help through cobra instead of printing it directly.")
		})
	}
}

// captureExecuteHelp runs the help path Execute takes for args.
func captureExecuteHelp(t *testing.T, args []string) string {
	t.Helper()

	rootCmd, _, err := BuildRoot(log.GetInstance())
	assert.NilError(t, err)

	requested, helpCmd := helpRequested(rootCmd, args)
	assert.Assert(t, requested, "expected help to be requested for %v", args)

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	assert.NilError(t, executeHelp(context.Background(), rootCmd, helpCmd))

	return out.String()
}

// captureCobraHelp lets cobra parse the help flag itself, without the
// interception. args must be an invocation cobra answers with help only.
func captureCobraHelp(t *testing.T, args []string) string {
	t.Helper()

	rootCmd, _, err := BuildRoot(log.GetInstance())
	assert.NilError(t, err)

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs(args)
	_, err = rootCmd.ExecuteC()
	assert.NilError(t, err)

	return out.String()
}
