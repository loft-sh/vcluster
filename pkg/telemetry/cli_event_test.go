package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/loft-sh/analytics-client/client"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Plain alphanumeric on purpose: the properties are JSON inside JSON, and
// characters that get escaped would hide the value from a substring search.
const commandLineValue = "s3cr3tvalue"

// runDuration is how long every test command runs, so the recorded duration
// has a lower bound only a start taken before the run can reach.
const runDuration = 20 * time.Millisecond

type recordingAnalyticsClient struct {
	events []client.Event
}

func (r *recordingAnalyticsClient) RecordEvent(event client.Event) {
	r.events = append(r.events, event)
}

func (r *recordingAnalyticsClient) Flush() {}

// textValue is a flag value that keeps whatever text it is given and reports
// the type it is told to.
type textValue struct {
	text     string
	typeName string
}

func (v *textValue) String() string { return v.text }

func (v *textValue) Set(text string) error {
	v.text = text
	return nil
}

func (v *textValue) Type() string { return v.typeName }

// testCLI is a command tree that declares flags the way the CLI commands do
// and starts the collector from the root pre-run, as the CLI does.
type testCLI struct {
	rootCmd         *cobra.Command
	cliConfig       *config.CLI
	analyticsClient *recordingAnalyticsClient
	collector       *cliCollector
}

func newTestCLI(configuredDriver string) *testCLI {
	cli := &testCLI{
		cliConfig:       &config.CLI{Driver: config.Driver{Type: config.DriverType(configuredDriver)}},
		analyticsClient: &recordingAnalyticsClient{},
	}

	var driver, chartVersion string
	var upgradeExisting bool

	// run fails with an error quoting the args and flag values, as the wrapped
	// errors of the real commands can, unless --succeed is set. Before that it
	// overwrites what the flags are bound to and the configured driver, as
	// commands do while they run.
	run := func(cmd *cobra.Command, args []string) error {
		flagValues := []string{}
		for _, name := range []string{"access-key", "driver", "chart-version", "namespace"} {
			if flag := cmd.Flags().Lookup(name); flag != nil {
				flagValues = append(flagValues, flag.Value.String())
			}
		}

		time.Sleep(runDuration)
		driver, chartVersion, upgradeExisting = string(config.PlatformDriver), "9.9.9", false
		cli.cliConfig.Driver.Type = config.PlatformDriver

		if succeed, _ := cmd.Flags().GetBool("succeed"); succeed {
			return nil
		}

		return fmt.Errorf("failed with args %q and flags %q", args, flagValues)
	}

	rootCmd := &cobra.Command{
		Use:           "vcluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			cli.collector, err = newCLICollector(cli.analyticsClient, cli.cliConfig, cmd)

			return err
		},
	}
	rootCmd.PersistentFlags().StringP("namespace", "n", "", "")
	rootCmd.PersistentFlags().Bool("succeed", false, "")

	loginCmd := &cobra.Command{Use: "login", RunE: run}
	loginCmd.Flags().String("access-key", "", "")

	createCmd := &cobra.Command{Use: "create", Aliases: []string{"new"}, RunE: run}
	createCmd.Flags().StringArray("set", nil, "")
	createCmd.Flags().StringArrayP("values", "f", nil, "")
	createCmd.Flags().StringVar(&driver, "driver", "", "")
	createCmd.Flags().StringVar(&chartVersion, "chart-version", "", "")
	createCmd.Flags().BoolVar(&upgradeExisting, "upgrade", false, "")
	createCmd.Flags().Bool("connect", true, "")

	connectCmd := &cobra.Command{Use: "connect", RunE: run}

	// flags that share a name or a type name with the ones whose value is sent
	lookalikeCmd := &cobra.Command{Use: "lookalike", RunE: run}
	lookalikeCmd.Flags().StringSlice("output", nil, "")
	lookalikeCmd.Flags().Count("version", "")
	lookalikeCmd.Flags().Var(&textValue{typeName: "bool"}, "force", "")
	lookalikeCmd.Flags().Var(&textValue{typeName: "string"}, "driver", "")

	platformCmd := &cobra.Command{Use: "platform"}
	platformLoginCmd := &cobra.Command{Use: "login", RunE: run}
	platformLoginCmd.Flags().String("access-key", "", "")
	platformCmd.AddCommand(platformLoginCmd)

	rootCmd.AddCommand(loginCmd, createCmd, connectCmd, lookalikeCmd, platformCmd)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	cli.rootCmd = rootCmd

	return cli
}

func TestRecordCLISendsOnlyBoundedValues(t *testing.T) {
	type properties = map[string]interface{}
	list := func(values ...interface{}) []interface{} { return append([]interface{}{}, values...) }

	testCases := []struct {
		name             string
		args             []string
		configuredDriver string
		// expected is nil when the command line never reaches the root pre-run,
		// so no collector exists and nothing is recorded.
		expected properties
		// cobraRun is set when cobra answers with its own function instead of
		// run, which does not take runDuration.
		cobraRun bool
	}{
		{
			name:             "flag value after equals sign",
			args:             []string{"login", "--access-key=" + commandLineValue, "https://" + commandLineValue},
			configuredDriver: "platform",
			expected: properties{
				"command": list("vcluster", "login"), "flags": list("access-key"), "flag_values": properties{},
				"args": 1.0, "driver": "platform", "error": "other",
			},
		},
		{
			name: "flag value as separate arg",
			args: []string{"login", "--access-key", commandLineValue},
			expected: properties{
				"command": list("vcluster", "login"), "flags": list("access-key"), "flag_values": properties{},
				"args": 0.0, "driver": "helm", "error": "other",
			},
		},
		{
			name: "repeated key value flag",
			args: []string{"create", "my-vcluster", "--set", "dsn=postgres://user:" + commandLineValue + "@host/db", "--set", commandLineValue + "=true"},
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("set"), "flag_values": properties{},
				"args": 1.0, "driver": "helm", "error": "other",
			},
		},
		{
			name: "short flags, separate and attached, through an alias",
			args: []string{"new", "my-vcluster", "-n", commandLineValue, "-f" + commandLineValue},
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("namespace", "values"), "flag_values": properties{},
				"args": 1.0, "driver": "helm", "error": "other",
			},
		},
		{
			name: "persistent flag before the command",
			args: []string{"-n", commandLineValue, "create", commandLineValue},
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("namespace"), "flag_values": properties{},
				"args": 1.0, "driver": "helm", "error": "other",
			},
		},
		{
			name: "passthrough command after the terminator",
			args: []string{"connect", "my-vcluster", "--", "kubectl", "create", "secret", "generic", "db", "--from-literal=password=" + commandLineValue},
			expected: properties{
				"command": list("vcluster", "connect"), "flags": list(), "flag_values": properties{},
				"args": 7.0, "driver": "helm", "error": "other",
			},
		},
		{
			name:             "listed and boolean flags keep the values that were typed",
			args:             []string{"create", "my-vcluster", "--driver", "docker", "--chart-version=v0.30.0-rc.1", "--upgrade", "--connect=false", "--succeed"},
			configuredDriver: "helm",
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("chart-version", "connect", "driver", "succeed", "upgrade"),
				"flag_values": properties{"chart-version": "v0.30.0-rc.1", "connect": "false", "driver": "docker", "succeed": "true", "upgrade": "true"},
				"args":        1.0, "driver": "docker",
			},
		},
		{
			name:             "listed flags and config with values outside their set",
			args:             []string{"create", "my-vcluster", "--driver", commandLineValue, "--chart-version", "0.30.0-" + commandLineValue},
			configuredDriver: commandLineValue,
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("chart-version", "driver"), "flag_values": properties{},
				"args": 1.0, "error": "other",
			},
		},
		{
			name:             "invalid driver flag is not replaced by the configured one",
			args:             []string{"create", "my-vcluster", "--driver", commandLineValue},
			configuredDriver: "helm",
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("driver"), "flag_values": properties{},
				"args": 1.0, "error": "other",
			},
		},
		{
			name:             "invalid configured driver",
			args:             []string{"create", "my-vcluster", "--succeed"},
			configuredDriver: commandLineValue,
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("succeed"), "flag_values": properties{"succeed": "true"},
				"args": 1.0,
			},
		},
		{
			name:             "empty driver flag with an invalid configured driver",
			args:             []string{"create", "my-vcluster", "--driver=", "--succeed"},
			configuredDriver: commandLineValue,
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("driver", "succeed"), "flag_values": properties{"succeed": "true"},
				"args": 1.0,
			},
		},
		{
			name:             "empty driver flag falls back to the configured one",
			args:             []string{"create", "my-vcluster", "--driver=", "--succeed"},
			configuredDriver: "docker",
			expected: properties{
				"command": list("vcluster", "create"), "flags": list("driver", "succeed"), "flag_values": properties{"succeed": "true"},
				"args": 1.0, "driver": "docker",
			},
		},
		{
			name: "flags that only look like the ones whose value is sent",
			args: []string{"lookalike", "--output", "json", "--version", "--force=" + commandLineValue, "--driver", commandLineValue},
			expected: properties{
				"command": list("vcluster", "lookalike"), "flags": list("driver", "force", "output", "version"), "flag_values": properties{},
				"args": 0.0, "error": "other",
			},
		},
		{
			name: "nested command",
			args: []string{"platform", "login", "--access-key", commandLineValue},
			expected: properties{
				"command": list("vcluster", "platform", "login"), "flags": list("access-key"), "flag_values": properties{},
				"args": 0.0, "driver": "helm", "error": "other",
			},
		},
		{
			name: "help command with a topic",
			args: []string{"help", commandLineValue},
			expected: properties{
				"command": list("vcluster", "help"), "flags": list(), "flag_values": properties{},
				"args": 1.0, "driver": "helm",
			},
			cobraRun: true,
		},
		{
			name: "unknown command",
			args: []string{commandLineValue, "--access-key", commandLineValue},
		},
		{
			name: "unknown flag",
			args: []string{"create", "--" + commandLineValue + "=" + commandLineValue},
		},
		{
			name: "flag without its value",
			args: []string{"login", commandLineValue, "--access-key"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			previousArgs := os.Args
			os.Args = append([]string{"/home/" + commandLineValue + "/vcluster"}, testCase.args...)
			t.Cleanup(func() { os.Args = previousArgs })

			cli := newTestCLI(testCase.configuredDriver)
			cli.rootCmd.SetArgs(testCase.args)
			_, err := cli.rootCmd.ExecuteC()

			if testCase.expected == nil {
				assert.Assert(t, err != nil)
				assert.Assert(t, cli.collector == nil, "collector started for a command line that never ran")
				return
			}

			cli.collector.RecordCLI(cli.cliConfig, nil, err)
			assert.Equal(t, len(cli.analyticsClient.events), 1)

			payload, marshalErr := json.Marshal(client.Request{Data: cli.analyticsClient.events})
			assert.NilError(t, marshalErr)
			assert.Assert(t, !strings.Contains(string(payload), commandLineValue), "payload carries a command line value: %s", payload)

			sent := properties{}
			assert.NilError(t, json.Unmarshal([]byte(cli.analyticsClient.events[0]["event"]["properties"].(string)), &sent))

			minDuration := float64(runDuration.Milliseconds())
			if testCase.cobraRun {
				minDuration = 0
			}
			duration, isNumber := sent["duration_ms"].(float64)
			assert.Assert(t, isNumber && duration >= minDuration && duration < 60_000, "duration_ms is %v", sent["duration_ms"])
			delete(sent, "duration_ms")

			testCase.expected["version"] = upgrade.GetVersion()
			assert.DeepEqual(t, sent, testCase.expected)
		})
	}
}

func TestSentFlagValues(t *testing.T) {
	testCases := map[string]struct {
		accepted []string
		rejected []string
	}{
		"driver":     {accepted: []string{"helm", "platform", "docker"}, rejected: []string{"Helm", "helm ", "kind"}},
		"use-driver": {accepted: []string{"helm", "platform", "docker"}, rejected: []string{"Docker", "docker,helm"}},
		"output":     {accepted: []string{"table", "json", "yaml"}, rejected: []string{"JSON", "jsonpath={.data}", "/tmp/out.json", "[json]"}},
		"chart-version": {
			accepted: []string{"0.30.0", "v0.30.0", "0.30.0-alpha.1", "v0.30.0-rc.12", "0.31.0-next.3"},
			rejected: []string{"latest", "0.30", "0.30.0+" + commandLineValue, "0.30.0-" + commandLineValue, "0.30.0-rc." + commandLineValue, "12345.0.0", "0.0.123456789", "v0.30.0\n", " 0.30.0"},
		},
		"version": {
			accepted: []string{"v0.30.0", "0.30.0-rc.1"},
			rejected: []string{"latest", "stable", "0.30.0+" + commandLineValue, "1"},
		},
	}

	// every listed flag needs a case here, so adding one without saying what
	// it must reject fails
	assert.Equal(t, len(testCases), len(sentFlagValues))

	alwaysRejected := []string{"", commandLineValue, "true", "[]", "a=b", "https://user:" + commandLineValue + "@host", strings.Repeat("1", 64)}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			isValid := sentFlagValues[name]
			assert.Assert(t, isValid != nil, "flag %s is not listed", name)

			for _, value := range testCase.accepted {
				assert.Assert(t, isValid(value), "%q should be sent", value)
			}
			for _, value := range append(testCase.rejected, alwaysRejected...) {
				assert.Assert(t, !isValid(value), "%q should not be sent", value)
			}
		})
	}
}

func TestErrorCategory(t *testing.T) {
	secretsResource := schema.GroupResource{Resource: "secrets"}
	wrap := func(err error) error {
		return fmt.Errorf("deploy %s: %w", commandLineValue, err)
	}

	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{name: "plain error", err: errors.New(commandLineValue), expected: "other"},
		{name: "vcluster not found", err: wrap(&find.VClusterNotFoundError{Name: commandLineValue}), expected: "vcluster_not_found"},
		{name: "kubernetes unauthorized", err: wrap(kerrors.NewUnauthorized(commandLineValue)), expected: "kubernetes_unauthorized"},
		{name: "kubernetes forbidden", err: wrap(kerrors.NewForbidden(secretsResource, commandLineValue, errors.New(commandLineValue))), expected: "kubernetes_forbidden"},
		{name: "kubernetes not found", err: wrap(kerrors.NewNotFound(secretsResource, commandLineValue)), expected: "kubernetes_not_found"},
		{name: "kubernetes already exists", err: wrap(kerrors.NewAlreadyExists(secretsResource, commandLineValue)), expected: "kubernetes_other"},
		{
			name:     "kubernetes reason invented by the server",
			err:      wrap(&kerrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReason(commandLineValue), Message: commandLineValue}}),
			expected: "kubernetes_other",
		},
		{name: "deadline", err: wrap(context.DeadlineExceeded), expected: "timeout"},
		{name: "canceled", err: wrap(context.Canceled), expected: "canceled"},
		{name: "connection refused", err: wrap(&url.Error{Op: "Get", URL: "https://" + commandLineValue, Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}}), expected: "network"},
		{name: "canceled inside a network error", err: wrap(&url.Error{Op: "Get", URL: "https://" + commandLineValue, Err: context.Canceled}), expected: "canceled"},
		{name: "deadline inside a network error", err: wrap(&url.Error{Op: "Get", URL: "https://" + commandLineValue, Err: context.DeadlineExceeded}), expected: "timeout"},
		{name: "name resolution", err: wrap(&net.DNSError{Err: "no such host", Name: commandLineValue}), expected: "network"},
		{name: "network timeout", err: wrap(&net.DNSError{Err: "i/o timeout", Name: commandLineValue, IsTimeout: true}), expected: "timeout"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, errorCategory(testCase.err), testCase.expected)
		})
	}
}
