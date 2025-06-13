package translator

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
)

func TestMatches(t *testing.T) {
	defaultVclusterNs := "vcluster"
	cases := []struct {
		name                  string
		mappings              map[string]string
		hostName              string
		hostNs                string
		virtualName           string
		virtualNs             string
		vClusterHostNamespace string
		noMatchExpected       bool
		expectedVirtual       types.NamespacedName
	}{
		{
			name: "mirror",
			mappings: map[string]string{
				"my-ns/my-cm":   "my-ns/my-cm",
				"my-ns/my-cm-2": "my-ns/my-cm-3",
				"my-ns-2/*":     "my-ns-2/*",
			},
			hostName:              "my-cm",
			hostNs:                "my-ns",
			virtualName:           "my-cm",
			virtualNs:             "my-ns",
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "my-cm", Namespace: "my-ns"},
		},
		{
			name: "match all in namespace",
			mappings: map[string]string{
				"my-ns/*":   "my-ns-2/*",
				"my-ns-3/*": "my-ns-3/*",
			},
			hostName:              "my-cm",
			hostNs:                "my-ns",
			virtualName:           "my-cm",
			virtualNs:             "my-ns-2",
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "my-cm", Namespace: "my-ns-2"},
		},
		{
			name: "change name and namespace",
			mappings: map[string]string{
				"my-ns/my-cm": "my-ns-2/my-cm-2",
				"my-ns-3/*":   "my-ns-3/*",
			},
			hostName:              "my-cm",
			hostNs:                "my-ns",
			virtualName:           "my-cm-2",
			virtualNs:             "my-ns-2",
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "my-cm-2", Namespace: "my-ns-2"},
		},
		{
			name: "all from vCluster host namespace to another namespace in virtual",
			mappings: map[string]string{
				"": "my-ns",
			},
			hostName:              "my-cm",
			hostNs:                "vcluster",
			virtualName:           "my-cm",
			virtualNs:             "my-ns",
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "my-cm", Namespace: "my-ns"},
		},
		{
			name: "no match",
			mappings: map[string]string{
				"":              "my-ns",
				"my-ns/*":       "my-ns-2/*",
				"my-ns-2/my-cm": "my-ns-2/my-cm",
			},
			hostName:              "my-cm-2",
			hostNs:                "my-ns-2",
			virtualName:           "",
			virtualNs:             "",
			noMatchExpected:       true,
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "", Namespace: ""}, // no match
		},
		{
			name: "kube-root-ca.crt skipped",
			mappings: map[string]string{
				"":              "my-ns",
				"my-ns/*":       "my-ns-2/*",
				"my-ns-2/my-cm": "my-ns-2/my-cm",
			},
			hostName:              "kube-root-ca.crt",
			hostNs:                "ingress-nginx",
			virtualName:           "",
			virtualNs:             "",
			noMatchExpected:       true,
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "", Namespace: ""}, // no match
		},
		{
			name: "a few objects from vcluster's namespace",
			mappings: map[string]string{
				"/my-cm":   "my-ns-2/my-cm-2",
				"/my-cm-2": "my-ns-3/my-cm-3",
			},
			hostName:              "my-cm",
			hostNs:                "vcluster",
			virtualName:           "my-cm-2",
			virtualNs:             "my-ns-2",
			vClusterHostNamespace: defaultVclusterNs,
			expectedVirtual:       types.NamespacedName{Name: "my-cm-2", Namespace: "my-ns-2"},
		},
		{
			name: "a few objects from vcluster's namespace (not default)",
			mappings: map[string]string{
				"/my-cm":   "my-ns-2/my-cm-2",
				"/my-cm-2": "my-ns-3/my-cm-3",
			},
			hostName:              "my-cm",
			hostNs:                "ns1",
			virtualName:           "my-cm-2",
			virtualNs:             "my-ns-2",
			vClusterHostNamespace: "ns1",
			expectedVirtual:       types.NamespacedName{Name: "my-cm-2", Namespace: "my-ns-2"},
		},
		{
			name: "vcluster's namespace match all but for several configmaps, sync to another namespace",
			mappings: map[string]string{
				"from-host-sync-test/*": "barfoo/*",
				"default/my-cm":         "barfoo/cm-my",
				"":                      "my-new-ns",
				"my-ns-4/*":             "my-ns-4/*",
				"/my-cm-5":              "my-virtual-namespace/my-cm-5",
				"/my-cm-6":              "my-virtual-namespace6/my-cm-6",
			},
			hostName:              "my-cm-6",
			hostNs:                "vcluster",
			virtualName:           "my-cm-6",
			virtualNs:             "my-virtual-namespace6",
			vClusterHostNamespace: "vcluster",
			expectedVirtual:       types.NamespacedName{Name: "my-cm-6", Namespace: "my-virtual-namespace6"},
		},
	}

	t.Run("match host", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, _ := matchesHostObject(&synccontext.SyncContext{}, tc.hostName, tc.hostNs, tc.mappings, tc.vClusterHostNamespace, func(hostName, _ string) bool {
					return hostName == "kube-root-ca.crt"
				})
				if got.Name == tc.virtualName && got.Namespace == tc.virtualNs {
					return
				}
				t.Logf("expectedVirtual %s/%s, got %s", tc.virtualNs, tc.virtualName, got.String())
				t.Fail()
			})
		}
	})

	t.Run("match virtual", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				virtualToHost := make(map[string]string, len(tc.mappings))
				for host, virtual := range tc.mappings {
					virtualToHost[virtual] = host
				}
				got, match := matchesVirtualObject(tc.virtualNs, tc.virtualName, virtualToHost, tc.vClusterHostNamespace)
				if tc.noMatchExpected && match != tc.noMatchExpected {
					return
				}
				if got.Name == tc.hostName && got.Namespace == tc.hostNs {
					return
				}
				t.Logf("expectedHost %s/%s, got %s", tc.hostNs, tc.hostName, got.String())
				t.Fail()
			})
		}
	})
}
