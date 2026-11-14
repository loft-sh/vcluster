package cli

import (
	"errors"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/terminal"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSelectHelmTargets(t *testing.T) {
	const (
		hostContext   = "kind-kind-cluster"
		nestedContext = "vcluster_parent_vcluster-parent_kind-kind-cluster"
	)

	testCases := []struct {
		name                  string
		vClusters             []find.VCluster
		kubeContext           string
		expectedTargets       []string
		expectedSkippedParent []string
		expectedInKubeSystem  int
	}{
		{
			name:        "no vclusters",
			kubeContext: hostContext,
		},
		{
			name: "all in the current context",
			vClusters: []find.VCluster{
				{Name: "vc-a", Namespace: "vcluster-a", Context: hostContext},
				{Name: "vc-b", Namespace: "vcluster-b", Context: hostContext},
			},
			kubeContext:     hostContext,
			expectedTargets: []string{"vc-a", "vc-b"},
		},
		{
			name: "nested context skips the parent vclusters",
			vClusters: []find.VCluster{
				{Name: "child", Namespace: "vcluster-child", Context: nestedContext},
				{Name: "parent", Namespace: "vcluster-parent", Context: hostContext},
				{Name: "sibling", Namespace: "vcluster-sibling", Context: hostContext},
			},
			kubeContext:           nestedContext,
			expectedTargets:       []string{"child"},
			expectedSkippedParent: []string{"parent", "sibling"},
		},
		{
			name: "counts targets in kube-system but not skipped ones",
			vClusters: []find.VCluster{
				{Name: "vc-sys", Namespace: metav1.NamespaceSystem, Context: hostContext},
				{Name: "vc-a", Namespace: "vcluster-a", Context: hostContext},
				{Name: "parent-sys", Namespace: metav1.NamespaceSystem, Context: "other-context"},
			},
			kubeContext:           hostContext,
			expectedTargets:       []string{"vc-sys", "vc-a"},
			expectedSkippedParent: []string{"parent-sys"},
			expectedInKubeSystem:  1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := selectHelmTargets(testCase.vClusters, testCase.kubeContext)

			assert.DeepEqual(t, names(result.Targets), testCase.expectedTargets)
			assert.DeepEqual(t, names(result.SkippedParent), testCase.expectedSkippedParent)
			assert.Equal(t, result.InKubeSystem, testCase.expectedInKubeSystem)
		})
	}
}

func names(vClusters []find.VCluster) []string {
	var result []string
	for _, vCluster := range vClusters {
		result = append(result, vCluster.Name)
	}
	return result
}

func TestConfirmDeleteAll(t *testing.T) {
	originalIsTerminalIn := terminal.IsTerminalIn
	t.Cleanup(func() {
		terminal.IsTerminalIn = originalIsTerminalIn
	})

	t.Run("non-interactive skips the prompt", func(t *testing.T) {
		terminal.IsTerminalIn = false

		confirmed, err := confirmDeleteAll(3, &DeleteOptions{NonInteractive: true}, log.Discard)
		assert.NilError(t, err)
		assert.Equal(t, confirmed, true)
	})

	t.Run("no terminal and no flag refuses", func(t *testing.T) {
		terminal.IsTerminalIn = false

		confirmed, err := confirmDeleteAll(3, &DeleteOptions{}, log.Discard)
		assert.ErrorContains(t, err, "--non-interactive")
		assert.Equal(t, confirmed, false)
	})
}

func TestReportDeleteResults(t *testing.T) {
	t.Run("all deleted", func(t *testing.T) {
		err := reportDeleteResults([]deleteResult{
			{Name: "vc-a", Location: "vcluster-a"},
			{Name: "vc-b", Location: "vcluster-b"},
		}, "NAMESPACE", log.Discard)
		assert.NilError(t, err)
	})

	t.Run("no results", func(t *testing.T) {
		err := reportDeleteResults(nil, "NAMESPACE", log.Discard)
		assert.NilError(t, err)
	})

	t.Run("some failed", func(t *testing.T) {
		failure := errors.New("helm delete failed")
		err := reportDeleteResults([]deleteResult{
			{Name: "vc-a", Location: "vcluster-a"},
			{Name: "vc-b", Location: "vcluster-b", Err: failure},
			{Name: "vc-c", Location: "vcluster-c", Err: errors.New("namespace is terminating")},
		}, "NAMESPACE", log.Discard)
		assert.ErrorContains(t, err, "2 of 3 virtual cluster(s) could not be deleted")
		assert.ErrorContains(t, err, "vcluster-b/vc-b: helm delete failed")
		assert.ErrorContains(t, err, "vcluster-c/vc-c: namespace is terminating")
		assert.Assert(t, errors.Is(err, failure), "wrapped error should be preserved")
	})

	t.Run("docker results have no location", func(t *testing.T) {
		err := reportDeleteResults([]deleteResult{
			{Name: "dock-a", Err: errors.New("container busy")},
		}, "", log.Discard)
		assert.ErrorContains(t, err, "1 of 1 virtual cluster(s) could not be deleted")
		assert.ErrorContains(t, err, "dock-a: container busy")
	})
}
