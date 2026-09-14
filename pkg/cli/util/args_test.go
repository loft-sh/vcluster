package util

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestNamedPositionalArgsValidator(t *testing.T) {
	// loop through a generated variety of inputs: arg counts, expected arg counts, and failMissing
	// since it depends on the numbers, it's easier to loop than writing a testable
	maxExpectedArgCount := 5
	maxActualArgsCount := maxExpectedArgCount + 5
	expectedArgs := []string{}
	testNum := 0
	// loop through maxExpectedArgCount lengths of expectedArgs
	for len(expectedArgs) <= maxExpectedArgCount {
		actualArgs := []string{}
		// loop through maxActualArgCount lengths of actualArgs
		for len(actualArgs) <= maxActualArgsCount {
			defer func() {
				panicErr := recover()
				if panicErr != nil {
					t.Fatalf("this function should never panic: %+v", panicErr)
				}
			}()
			testNum++
			// loop through both values of failMissing
			for _, failMissing := range []bool{true, false} {
				for _, failExtra := range []bool{true, false} {
					// execute test
					t.Logf("running test #%d with failMissing %v, failExtra %v, expectedArgs: %q, args: %q", testNum, failMissing, failExtra, expectedArgs, actualArgs)
					// if testNum == 23 {
					// 	t.Log("focus a test number number for debugging")
					// }
					_, validator := NamedPositionalArgsValidator(failMissing, failExtra, expectedArgs...)
					err := validator(&cobra.Command{}, actualArgs)
					if len(actualArgs) > len(expectedArgs) && failExtra {
						assert.ErrorContains(t, err, "extra arguments:", "expect error to not be nil as arg count is mismatched")
					} else if len(actualArgs) < len(expectedArgs) && failMissing {
						assert.ErrorContains(t, err, "please specify missing:", "expect error to not be nil as arg count is mismatched")
					} else {
						assert.NilError(t, err, "expect error to be nil as all args provided and no extra")
					}
					// append to actual args
					actualArgs = append(actualArgs, fmt.Sprintf("ARG_%d", len(actualArgs)))
				}
			}
		}
		// append to expected args
		expectedArgs = append(expectedArgs, fmt.Sprintf("ARG_NAME_%d", len(expectedArgs)))
	}
}
