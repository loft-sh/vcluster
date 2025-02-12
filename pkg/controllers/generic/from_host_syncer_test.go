package generic

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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseHostNamespacesFromMappings(tc.mappings, tc.vClusterNs)
			if len(got) != len(tc.expected) {
				t.Logf("expectedVirtual %d namespaces, got %d", len(tc.expected), len(got))
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
