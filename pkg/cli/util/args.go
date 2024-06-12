package util

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	NamespaceNameOnlyUseLine   string
	NamespaceNameOnlyValidator cobra.PositionalArgs

	VClusterNameOnlyUseLine string

	VClusterNameOnlyValidator cobra.PositionalArgs
)

func init() {
	NamespaceNameOnlyUseLine, NamespaceNameOnlyValidator = NamedPositionalArgsValidator(true, true, "NAMESPACE_NAME")
	VClusterNameOnlyUseLine, VClusterNameOnlyValidator = NamedPositionalArgsValidator(true, true, "VCLUSTER_NAME")
}

// NamedPositionalArgsValidator returns a cobra.PositionalArgs that returns a helpful
// error message if the arg number doesn't match.
// It also returns a string that can be appended to the cobra useline
//
// Example output for extra arguments with :
//
//	$ command arg asdf
//	[fatal]  command ARG_1 [flags]
//	Invalid Args: received 2 arguments, expected 1, extra arguments: "asdf"
//	Run with --help for more details
//
// Example output for missing arguments:
//
//	$ command
//	[fatal]  command ARG_1 [flags]
//	Invalid Args: received 0 arguments, expected 1, please specify missing: "ARG_!"
//	Run with --help for more details on arguments
func NamedPositionalArgsValidator(failMissing, failExtra bool, expectedArgs ...string) (string, cobra.PositionalArgs) {
	return " " + strings.Join(expectedArgs, " "), func(cmd *cobra.Command, args []string) error {
		numExpectedArgs := len(expectedArgs)
		numArgs := len(args)
		numMissing := numExpectedArgs - numArgs

		if numMissing == 0 {
			return nil
		}

		// didn't receive as many arguments as expected
		if numMissing > 0 && failMissing {
			// the last numMissing expectedArgs
			missingKeys := strings.Join(expectedArgs[len(expectedArgs)-(numMissing):], ", ")
			return fmt.Errorf("%s\nInvalid Args: received %d arguments, expected %d, please specify missing: %q\nRun with --help for more details on arguments", cmd.UseLine(), numArgs, numExpectedArgs, missingKeys)
		}

		// received more than expected
		if numMissing < 0 && failExtra {
			// received more than expected
			numExtra := -numMissing
			// the last numExtra args
			extraValues := strings.Join(args[len(args)-numExtra:], ", ")
			return fmt.Errorf("%s\nInvalid Args: received %d arguments, expected %d, extra arguments: %q\nRun with --help for more details on arguments", cmd.UseLine(), numArgs, numExpectedArgs, extraValues)
		}

		return nil
	}
}
