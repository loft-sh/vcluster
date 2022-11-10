package context

import (
	"testing"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestEnableControllers(t *testing.T) {
	testTable := []struct {
		desc           string
		optsModifier   func(*VirtualClusterOptions)
		expectEnabled  []string
		expectDisabled []string
		expectError    bool
		errSubString   string
		pause          bool
	}{
		{
			desc:           "default case",
			optsModifier:   func(v *VirtualClusterOptions) {},
			expectEnabled:  DefaultEnabledControllers.List(),
			expectDisabled: ExistingControllers.Difference(DefaultEnabledControllers).List(),
			expectError:    false,
		},
		{
			desc: "scheduler with pvc enabled, nodes not enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims"}
				v.EnableScheduler = true
			},
			expectError: true,
		},
		{
			desc: "scheduler with pvc enabled, storageclasses not enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims", "nodes"}
				v.EnableScheduler = true
			},
			expectEnabled:  append([]string{"hoststorageclasses"}, schedulerRequiredControllers.List()...),
			expectDisabled: []string{"storageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler with pvc enabled, storageclasses enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims", "nodes", "storageclasses"}
				v.EnableScheduler = true
			},
			expectEnabled:  append([]string{"storageclasses"}, schedulerRequiredControllers.List()...),
			expectDisabled: []string{"hoststorageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler with pvc enabled, hoststorageclasses enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims", "nodes"}
				v.EnableScheduler = true
			},
			expectEnabled:  append([]string{"hoststorageclasses"}, schedulerRequiredControllers.List()...),
			expectDisabled: []string{"storageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler disabled, storageclasses not enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims"}
			},
			expectEnabled:  []string{},
			expectDisabled: append([]string{"storageclasses", "hoststorageclasses"}, schedulerRequiredControllers.List()...),
			expectError:    false,
		},
		{
			desc: "storageclasses and hoststorageclasses enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"storageclasses", "hoststorageclasses"}
			},
			expectEnabled:  []string{},
			expectDisabled: []string{},
			expectError:    true,
		},
		{
			desc: "syncAllNodes true, nodes not enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{}
				v.SyncAllNodes = true
			},
			expectEnabled:  []string{},
			expectDisabled: []string{},
			expectError:    true,
			pause:          true,
		},
		{
			desc: "syncAllNodes true, nodes enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"nodes"}
				v.SyncAllNodes = true
			},
			expectEnabled:  []string{},
			expectDisabled: []string{},
			expectError:    false,
		},
	}

	for _, tc := range testTable {
		if tc.pause {
			t.Log("you can put a breakpoint here")
		}
		var opts VirtualClusterOptions
		AddFlags(pflag.NewFlagSet("test", pflag.PanicOnError), &opts)
		t.Logf("test case: %q", tc.desc)
		if tc.optsModifier != nil {
			tc.optsModifier(&opts)
		}
		foundControllers, err := parseControllers(&opts)
		if tc.expectError {
			assert.ErrorContains(t, err, tc.errSubString, "should have failed validation")
		} else {
			assert.NilError(t, err, "should have passed validation")
		}

		expectedNotFound := sets.NewString(tc.expectEnabled...).Difference(foundControllers)
		assert.Assert(t, is.Len(expectedNotFound.List(), 0), "should be enabled, but not enabled")

		disabledFound := sets.NewString(tc.expectDisabled...).Intersection(foundControllers)
		assert.Assert(t, is.Len(disabledFound.List(), 0), "should be disabled, but found enabled")

	}

}
