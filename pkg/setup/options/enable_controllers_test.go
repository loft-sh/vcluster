package options

import (
	"testing"

	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	fakeDiscovery "k8s.io/client-go/discovery/fake"
	clientTesting "k8s.io/client-go/testing"
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
			expectEnabled:  sets.List(DefaultEnabledControllers),
			expectDisabled: sets.List(ExistingControllers.Difference(DefaultEnabledControllers)),
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
			expectEnabled:  append([]string{"hoststorageclasses"}, sets.List(schedulerRequiredControllers)...),
			expectDisabled: []string{"storageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler with pvc enabled, storageclasses enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims", "nodes", "storageclasses"}
				v.EnableScheduler = true
			},
			expectEnabled:  append([]string{"storageclasses"}, sets.List(schedulerRequiredControllers)...),
			expectDisabled: []string{"hoststorageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler with pvc enabled, hoststorageclasses enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims", "nodes"}
				v.EnableScheduler = true
			},
			expectEnabled:  append([]string{"hoststorageclasses"}, sets.List(schedulerRequiredControllers)...),
			expectDisabled: []string{"storageclasses"},
			expectError:    false,
		},
		{
			desc: "scheduler disabled, storageclasses not enabled",
			optsModifier: func(v *VirtualClusterOptions) {
				v.Controllers = []string{"persistentvolumeclaims"}
			},
			expectEnabled:  []string{},
			expectDisabled: append([]string{"storageclasses", "hoststorageclasses"}, sets.List(schedulerRequiredControllers)...),
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

		foundControllers, err := ParseControllers(&opts)
		if tc.expectError {
			assert.ErrorContains(t, err, tc.errSubString, "should have failed validation")
		} else {
			assert.NilError(t, err, "should have passed validation")
		}

		expectedNotFound := sets.New(tc.expectEnabled...).Difference(foundControllers)
		assert.Assert(t, is.Len(sets.List(expectedNotFound), 0), "should be enabled, but not enabled")

		disabledFound := sets.New(tc.expectDisabled...).Intersection(foundControllers)
		assert.Assert(t, is.Len(sets.List(disabledFound), 0), "should be disabled, but found enabled")
	}
}

var csiStorageCapacityV1 = metav1.APIResource{
	Name:         "csistoragecapacities",
	SingularName: "csistoragecapacity",
	Namespaced:   false,
	Group:        "storage.k8s.io",
	Version:      "v1",
	Kind:         "CSIStorageCapacity",
}

var csiDriverV1 = metav1.APIResource{
	Name:         "csidrivers",
	SingularName: "csidriver",
	Namespaced:   false,
	Group:        "storage.k8s.io",
	Version:      "v1",
	Kind:         "CSIDriver",
}

var csiNodeV1 = metav1.APIResource{
	Name:         "csinodes",
	SingularName: "csinode",
	Namespaced:   false,
	Group:        "storage.k8s.io",
	Version:      "v1",
	Kind:         "CSINode",
}

func TestDisableMissingAPIs(t *testing.T) {
	tests := []struct {
		name             string
		apis             map[string][]metav1.APIResource
		expectedNotFound sets.Set[string]
		expectedFound    sets.Set[string]
	}{
		{
			name:             "K8s 1.21 or lower",
			apis:             map[string][]metav1.APIResource{},
			expectedNotFound: schedulerRequiredControllers,
			expectedFound:    sets.New[string](),
		},
		{
			name: "K8s 1.23 or lower",
			apis: map[string][]metav1.APIResource{
				storageV1GroupVersion: {csiNodeV1, csiDriverV1},
			},
			expectedNotFound: sets.New("csistoragecapacities"),
			expectedFound:    sets.New("csinodes", "csidrivers"),
		},
		{
			name: "K8s 1.24 or higher",
			apis: map[string][]metav1.APIResource{
				storageV1GroupVersion: {csiNodeV1, csiDriverV1, csiStorageCapacityV1},
			},
			expectedNotFound: sets.New[string](),
			expectedFound:    sets.New("csistoragecapacities", "csinodes", "csidrivers"),
		},
	}

	for i, testCase := range tests {
		t.Logf("running test #%d: %q", i, testCase.name)
		// initialize mocked discovery
		resourceLists := []*metav1.APIResourceList{}
		for groupVersion, resourceList := range testCase.apis {
			resourceLists = append(resourceLists, &metav1.APIResourceList{GroupVersion: groupVersion, APIResources: resourceList})
		}
		fakeDisoveryClient := &fakeDiscovery.FakeDiscovery{Fake: &clientTesting.Fake{Resources: resourceLists}}

		// run function
		actualControllers, err := DisableMissingAPIs(fakeDisoveryClient, ExistingControllers.Clone())
		assert.NilError(t, err)

		// unexpectedly not disabled
		notDisabled := actualControllers.Intersection(testCase.expectedNotFound).UnsortedList()
		assert.Assert(t, is.Len(notDisabled, 0), "expected %q to be disabled", testCase.expectedNotFound.UnsortedList())

		// should be enabled
		missing := testCase.expectedFound.Difference(actualControllers).UnsortedList()
		assert.Assert(t, is.Len(missing, 0), "expected %q to be found, but found only: %q", testCase.expectedFound.UnsortedList(), actualControllers.Intersection(testCase.expectedFound).UnsortedList())
	}
}
