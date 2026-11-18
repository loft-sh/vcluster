package telemetry

import (
	"cmp"
	"context"
	"errors"
	"net"
	"regexp"
	"slices"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// releaseVersion matches release versions such as v0.30.0 or 1.33.1-rc.2. It
// is deliberately narrower than semver: an arbitrary pre-release or build
// suffix could hold any text.
var releaseVersion = regexp.MustCompile(`^v?\d{1,4}\.\d{1,4}\.\d{1,4}(-(alpha|beta|rc|next)\.\d{1,4})?$`)

func oneOf(allowed ...string) func(string) bool {
	return func(value string) bool {
		return slices.Contains(allowed, value)
	}
}

// isDriver accepts the names the CLI accepts for a driver. The parser also
// takes an empty string, which is not a value that was typed.
func isDriver(value string) bool {
	_, err := config.ParseDriverType(value)
	return value != "" && err == nil
}

// sentFlagValues lists, by flag name, the non-boolean flags whose value is
// sent, and the check a value has to pass. A value that fails the check is
// left out and only the flag name is sent. Flags that are not listed here
// never have their value sent, so a flag belongs here only if every value
// its check accepts is safe to send for every command that has a flag of
// that name.
var sentFlagValues = map[string]func(string) bool{
	"driver":        isDriver,
	"use-driver":    isDriver,
	"output":        oneOf("table", "json", "yaml"),
	"chart-version": releaseVersion.MatchString,
	"version":       releaseVersion.MatchString,
}

// cliCommandProperties describes the invocation of cmd without any free-form
// command line value: the command path, the names of the flags that were set,
// the values of boolean flags and of the flags in sentFlagValues, the number of
// positional args and the driver the command starts with.
func cliCommandProperties(cliConfig *config.CLI, cmd *cobra.Command) map[string]interface{} {
	flagNames := []string{}
	flagValues := map[string]string{}
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		flagNames = append(flagNames, flag.Name)

		isValid := sentFlagValues[flag.Name]
		if flag.Value.Type() == "bool" {
			isValid = oneOf("true", "false")
		}
		if value := flag.Value.String(); isValid != nil && isValid(value) {
			flagValues[flag.Name] = value
		}
	})

	properties := map[string]interface{}{
		"command":     strings.Fields(cmd.CommandPath()),
		"flags":       flagNames,
		"flag_values": flagValues,
		"args":        len(cmd.Flags().Args()),
	}

	// resolved the way the commands resolve it: a non-empty --driver wins over
	// the config, and nothing at all means helm
	driverFlag := ""
	if flag := cmd.Flags().Lookup("driver"); flag != nil {
		driverFlag = flag.Value.String()
	}
	if driver, err := config.ParseDriverType(cmp.Or(driverFlag, string(cliConfig.Driver.Type))); err == nil {
		properties["driver"] = string(driver)
	}

	return properties
}

// errorCategory names the kind of failure behind err from a fixed set. It
// looks only at error types and sentinel values in the chain, never at text.
func errorCategory(err error) string {
	var (
		notFoundErr *find.VClusterNotFoundError
		apiStatus   kerrors.APIStatus
		netErr      net.Error
	)

	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.As(err, &notFoundErr):
		return "vcluster_not_found"
	case kerrors.IsUnauthorized(err):
		return "kubernetes_unauthorized"
	case kerrors.IsForbidden(err):
		return "kubernetes_forbidden"
	case kerrors.IsNotFound(err):
		return "kubernetes_not_found"
	case errors.As(err, &apiStatus):
		return "kubernetes_other"
	case errors.As(err, &netErr):
		if netErr.Timeout() {
			return "timeout"
		}
		return "network"
	default:
		return "other"
	}
}
