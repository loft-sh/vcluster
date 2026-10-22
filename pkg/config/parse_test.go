package config

import (
	"sort"
	"testing"
)

func TestParseHostNamespacesFromMappings(t *testing.T) {
	cases := []struct {
		name       string
		vClusterNs string
		mappings   map[string]string
		expected   []string
	}{
		{
			name:       "one namespace and wildcards",
			vClusterNs: "vcluster",
			mappings: map[string]string{
				"":            "target/",
				"my-ns/my-cm": "target2/my-cm",
				"my-ns-2/*":   "target3/*",
			},
			expected: []string{"vcluster", "my-ns", "my-ns-2"},
		},
		{
			name:       "no namespaces",
			vClusterNs: "vcluster",
			mappings:   map[string]string{},
			expected:   []string{},
		},
		{
			name:       "vcluster namespace different than vCluster",
			vClusterNs: "ns1",
			mappings: map[string]string{
				"/my-cm-1": "default/my-virtual-1",
				"/my-cm-2": "default/my-virtual-2",
			},
			expected: []string{"ns1"},
		},
		{
			name:       "repeated host namespaces",
			vClusterNs: "vcluster",
			mappings: map[string]string{
				"my-ns/my-cm":   "target2/my-cm",
				"my-ns/my-cm-2": "target3/my-cm-2",
			},
			expected: []string{"my-ns"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseHostNamespacesFromMappings(tc.mappings, tc.vClusterNs)
			if len(got) != len(tc.expected) {
				t.Logf("expectedVirtual %d namespaces (%v), got %d (%v)", len(tc.expected), tc.expected, len(got), got)
				t.Fail()
			}
			sort.Strings(got)
			sort.Strings(tc.expected)
			for i := range got {
				if tc.expected[i] != got[i] {
					t.Logf("expectedVirtual %s, got %s", tc.expected[i], got[i])
					t.Fail()
				}
			}
		})
	}
}
