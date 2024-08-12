package patches

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	patchesregex "github.com/loft-sh/vcluster/pkg/patches/regex"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type patchTestCase struct {
	name  string
	patch *config.Patch

	obj1 string
	obj2 string

	nameResolver NameResolver
	expected     string
	expectedErr  error
}

// TODO update tests

func TestPatch(t *testing.T) {
	True := true

	testCases := []*patchTestCase{
		{
			name: "copy merge",
			patch: &config.Patch{
				Operation: config.PatchTypeCopyFromObject,
				FromPath:  "status.test",
				Path:      "test",
			},
			obj1: `spec: {}
test:
    abc: def`,
			obj2: `status:
    test: test`,
			expected: `spec: {}
test: test`,
		},
		{
			name: "copy",
			patch: &config.Patch{
				Operation: config.PatchTypeCopyFromObject,
				FromPath:  "status",
				Path:      "status",
			},
			obj1: `spec: {}`,
			obj2: `status:
    test: test`,
			expected: `spec: {}
status:
    test: test`,
		},
		{
			name: "simple",
			patch: &config.Patch{
				Operation: config.PatchTypeReplace,
				Path:      "test.test2",
				Value:     "abc",
			},
			obj1: `test:
    test2: def`,
			expected: `test:
    test2: abc`,
		},
		{
			name: "insert",
			patch: &config.Patch{
				Operation: config.PatchTypeAdd,
				Path:      "test.test2[0].test3",
				Value:     "abc",
			},
			obj1: `test:
    test3: {}
test2: {}`,
			expected: `test:
    test3: {}
    test2:
        - test3: abc
test2: {}`,
		},
		{
			name: "insert slice",
			patch: &config.Patch{
				Operation: config.PatchTypeAdd,
				Path:      "test.test2",
				Value:     "abc",
			},
			obj1: `test:
    test2:
        - test`,
			expected: `test:
    test2:
        - test
        - abc`,
		},
		{
			name: "insert slice",
			patch: &config.Patch{
				Operation: config.PatchTypeReplace,
				Path:      "test..abc",
				Value:     "def",
			},
			obj1: `test:
    test2:
        - abc: test
        - abc: test2`,
			expected: `test:
    test2:
        - abc: def
        - abc: def`,
		},
		{
			name: "condition",
			patch: &config.Patch{
				Operation: config.PatchTypeReplace,
				Path:      "test.abc",
				Value:     "def",
				Conditions: []*config.PatchCondition{
					{
						Path:  "test.status",
						Empty: &True,
					},
				},
			},
			obj1: `test:
    abc: test`,
			expected: `test:
    abc: def`,
		},
		{
			name: "condition equal",
			patch: &config.Patch{
				Operation: config.PatchTypeReplace,
				Path:      "test.abc",
				Value:     "def",
				Conditions: []*config.PatchCondition{
					{
						Path: "test.status",
						Equal: map[string]interface{}{
							"test": "test",
						},
					},
				},
			},
			obj1: `test:
    status:
        test: test
    abc: test`,
			expected: `test:
    status:
        test: test
    abc: def`,
		},
		{
			name: "condition equal",
			patch: &config.Patch{
				Operation: config.PatchTypeReplace,
				Path:      "test.abc",
				Value:     "def",
				Conditions: []*config.PatchCondition{
					{
						Path: "test.status",
						Equal: map[string]interface{}{
							"test": "test1",
						},
					},
				},
			},
			obj1: `test:
    status:
        test: test
    abc: test`,
			expected: `test:
    status:
        test: test
    abc: test`,
		},
		{
			name: "resolve label selector",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteLabelSelector,
				Path:      "test.abc",
			},
			nameResolver: &fakeNameResolver{},
			obj1: `test:
    abc: {}`,
			expected: `test:
    abc:
        test: test`,
		},
		{
			name: "resolve empty label selector",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteLabelSelector,
				Path:      "test.abc",
			},
			nameResolver: &fakeNameResolver{},
			obj1: `test:
    abc: null`,
			expected: `test:
    abc: null`,
		},
		{
			name: "resolve filled label selector",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteLabelSelector,
				Path:      "test.abc",
			},
			nameResolver: &fakeNameResolver{},
			obj1: `test:
    abc:
        test123: test123`,
			expected: `test:
    abc:
        test: test
        test123: test123`,
		},
		{
			name: "rewrite name",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteName,
				Path:      "name",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1:     `name: abc`,
			expected: fmt.Sprint(`name: abc-x-default-x-`, translate.VClusterName),
		},
		{
			name: "rewrite name - invalid object",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteName,
				Path:      "name{]",
			},
			obj1:        `name: {}`,
			expectedErr: errors.New("parsing path"),
		},
		{
			name: "rewrite name - namespace based",
			patch: &config.Patch{
				Operation:     config.PatchTypeRewriteName,
				Path:          "root.list",
				NamePath:      "nm",
				NamespacePath: "ns",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1: `root:
  list:
    - nm: abc
      ns: pqr
    - nm: def
      ns: xyz`,
			expected: `root:
    list:
        - nm: abc-x-pqr-x-` + fmt.Sprint(translate.VClusterName) + `
          ns: vcluster
        - nm: def-x-xyz-x-` + fmt.Sprint(translate.VClusterName) + `
          ns: vcluster`,
		},
		{
			name: "rewrite name - multiple - no namespace",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteName,
				Path:      "root.list",
				NamePath:  "nm",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1: `root:
  list:
    - nm: abc
      ns: pqr
    - nm: def
      ns: pqr`,
			expected: `root:
    list:
        - nm: abc-x-default-x-` + fmt.Sprint(translate.VClusterName) + `
          ns: pqr
        - nm: def-x-default-x-` + fmt.Sprint(translate.VClusterName) + `
          ns: pqr`,
		},
		{
			name: "rewrite name - multiple name matches",
			patch: &config.Patch{
				Operation:     config.PatchTypeRewriteName,
				Path:          "root.includes",
				NamePath:      "names..nm",
				NamespacePath: "namespace",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1: `root:
  includes:
    - names:
        - nm: abc
        - nm: def
      namespace: pqr`,
			expected: `root:
    includes:
        - names:
            - nm: abc-x-pqr-x-` + fmt.Sprint(translate.VClusterName) + `
            - nm: def-x-pqr-x-` + fmt.Sprint(translate.VClusterName) + `
          namespace: vcluster`,
		},
		{
			name: "rewrite name - single name match - non array",
			patch: &config.Patch{
				Operation:     config.PatchTypeRewriteName,
				Path:          "root.includes",
				NamePath:      "nm",
				NamespacePath: "namespace",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1: `root:
  includes:
    nm: abc
    namespace: pqr`,
			expected: `root:
    includes:
        nm: abc-x-pqr-x-` + fmt.Sprint(translate.VClusterName) + `
        namespace: vcluster`,
		},
		{
			name: "rewrite name - multiple name matches - multiple namespace references",
			patch: &config.Patch{
				Operation:     config.PatchTypeRewriteName,
				Path:          "root.includes",
				NamePath:      "..names..nm",
				NamespacePath: "..namespaces..ns",
			},
			nameResolver: &fakeVirtualToHostNameResolver{
				namespace:       "default",
				targetNamespace: "vcluster",
			},
			obj1: `root:
  includes:
    - names:
        - nm: abc
        - nm: def
      namespaces:
        - ns: pqr
        - ns: xyz`,
			expectedErr: errors.New("found multiple namespace references"),
		},
		// 	{
		// 		name: "rewrite label key",
		// 		patch: &config.Patch{
		// 			Operation: config.PatchTypeRewriteLabelKey,
		// 			Path:      "test.label",
		// 		},
		// 		nameResolver: &fakeVirtualToHostNameResolver{},
		// 		obj1: `test:
		// label: myLabel`,
		// 		expected: `test:
		// label: vcluster.loft.sh/label-suffix-x-cb4e76426f`,
		// 	},
		// 	{
		// 		name: "rewrite label key - many",
		// 		patch: &config.Patch{
		// 			Operation: config.PatchTypeRewriteLabelKey,
		// 			Path:      "test.labels[*]",
		// 		},
		// 		nameResolver: &fakeVirtualToHostNameResolver{},
		// 		obj1: `test:
		// labels:
		//   - myLabel
		//   - myLabel2`,
		// 		expected: `test:
		// labels:
		//     - vcluster.loft.sh/label-suffix-x-cb4e76426f
		//     - vcluster.loft.sh/label-suffix-x-bae4a2c2e5`,
		// 	},
		{
			name: "rewrite name should not panic when match is not scalar",
			patch: &config.Patch{
				Operation: config.PatchTypeRewriteName,
				Path:      "test.endpoints[*]",
			},
			nameResolver: &fakeVirtualToHostNameResolver{},
			obj1: `test:
    endpoints:
      - name: abc
      - name: def`,
			expected: `test:
    endpoints:
        - name: abc
        - name: def`,
		},
	}

	for _, testCase := range testCases {
		obj1, err := NewNodeFromString(testCase.obj1)
		assert.NilError(t, err, "error in node creation in test case %s", testCase.name)

		var obj2 *yaml.Node
		if testCase.obj2 != "" {
			obj2, err = NewNodeFromString(testCase.obj2)
			assert.NilError(t, err, "error in node creation in test case %s", testCase.name)
		}

		err = applyPatch(obj1, obj2, testCase.patch, testCase.nameResolver)
		if testCase.expectedErr != nil {
			assert.ErrorContains(t, err, testCase.expectedErr.Error())
			continue
		}

		assert.NilError(t, err, "error in applying patch in test case %s", testCase.name)

		// compare output
		out, err := yaml.Marshal(obj1)

		assert.NilError(t, err, "error in yaml marshal in test case %s", testCase.name)
		assert.Equal(t, strings.TrimSpace(string(out)), testCase.expected, "error in comparison in test case %s", testCase.name)
	}
}

type fakeNameResolver struct{}

func (f *fakeNameResolver) TranslateName(name string, _ *regexp.Regexp, _ string) (string, error) {
	return name, nil
}

func (f *fakeNameResolver) TranslateNameWithNamespace(name string, _ string, _ *regexp.Regexp, _ string) (string, error) {
	return name, nil
}

func (f *fakeNameResolver) TranslateLabelKey(key string) (string, error) {
	return key, nil
}

var ErrNilSelector = errors.New("fake: nil selector")

func (f *fakeNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	if selector == nil {
		return nil, ErrNilSelector
	}

	if selector.MatchLabels == nil {
		selector.MatchLabels = map[string]string{}
	}
	selector.MatchLabels["test"] = "test"
	return selector, nil
}

func (f *fakeNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	if selector == nil {
		return nil, ErrNilSelector
	}
	selector["test"] = "test"
	return selector, nil
}

func (f *fakeNameResolver) TranslateNamespaceRef(string) (string, error) {
	return "default", nil
}

type fakeVirtualToHostNameResolver struct {
	namespace       string
	targetNamespace string
}

func (r *fakeVirtualToHostNameResolver) TranslateName(name string, regex *regexp.Regexp, _ string) (string, error) {
	return r.TranslateNameWithNamespace(name, r.namespace, regex, "")
}

func (r *fakeVirtualToHostNameResolver) TranslateNameWithNamespace(name string, namespace string, regex *regexp.Regexp, _ string) (string, error) {
	if regex != nil {
		return patchesregex.ProcessRegex(regex, name, func(name, ns string) types.NamespacedName {
			// if the regex match doesn't contain namespace - use the namespace set in this resolver
			if ns == "" {
				ns = namespace
			}
			return types.NamespacedName{Namespace: r.targetNamespace, Name: translate.Default.HostName(nil, name, ns).Name}
		}), nil
	}

	return translate.Default.HostName(nil, name, namespace).Name, nil
}

func (r *fakeVirtualToHostNameResolver) TranslateLabelKey(key string) (string, error) {
	return key, nil
}

func (r *fakeVirtualToHostNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	if selector == nil {
		return nil, ErrNilSelector
	}

	if selector.MatchLabels == nil {
		selector.MatchLabels = map[string]string{}
	}
	selector.MatchLabels["test"] = "test"
	return selector, nil
}

func (r *fakeVirtualToHostNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	if selector == nil {
		return nil, ErrNilSelector
	}
	selector["test"] = "test"
	return selector, nil
}

func (r *fakeVirtualToHostNameResolver) TranslateNamespaceRef(string) (string, error) {
	return r.targetNamespace, nil
}
