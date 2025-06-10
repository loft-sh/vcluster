package patch

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"
)

func Test(t *testing.T) {
	type test struct {
		Name string

		Object         string
		ExpectedObject string

		Adjust func(p Patch)
	}

	tests := []test{
		{
			Name: "Simple Translate",

			Object: `metadata:
  finalizers:
  - test
  name: test
  namespace: test`,

			ExpectedObject: `metadata:
  finalizers:
  - test-translated
  name: test-translated
  namespace: test`,

			Adjust: func(p Patch) {
				p.MustTranslate("metadata.finalizers[0]", func(_ string, val interface{}) (interface{}, error) {
					return val.(string) + "-translated", nil
				})
				p.MustTranslate("metadata.name", func(_ string, val interface{}) (interface{}, error) {
					return val.(string) + "-translated", nil
				})
			},
		},
		{
			Name: "Array",

			Object: `metadata:
  name: test
  namespace: test
  finalizers:
  - test1
  - test123`,

			ExpectedObject: `metadata:
  finalizers:
  - test1
  - test123-translated
  name: test
  namespace: test`,

			Adjust: func(p Patch) {
				p.MustTranslate("metadata.finalizers[1]", func(_ string, val interface{}) (interface{}, error) {
					return val.(string) + "-translated", nil
				})
			},
		},
		{
			Name: "Delete",

			Object: `metadata:
  name: test
  namespace: test
  finalizers:
  - test1
  - test123`,

			ExpectedObject: `metadata:
  name: test
  namespace: test`,

			Adjust: func(p Patch) {
				p.Delete("metadata.finalizers")
			},
		},
		{
			Name: "DeleteExcept",

			Object: `abc: abc
metadata:
  name: test
  namespace: test
  finalizers:
  - test1
  - test123`,

			ExpectedObject: `metadata:
  name: test`,

			Adjust: func(p Patch) {
				p.DeleteAllExcept("", "metadata")
				p.DeleteAllExcept("metadata", "name")
			},
		},
		{
			Name: "Value",

			Object: `metadata:
  name: name
  namespace: namespace`,

			ExpectedObject: `metadata:
  name: name
  namespace: name`,

			Adjust: func(p Patch) {
				name, _ := p.String("metadata.name")
				p.MustTranslate("metadata.namespace", func(_ string, _ interface{}) (interface{}, error) {
					return name, nil
				})
			},
		},
		{
			Name: "Set",

			Object: `metadata:
  name: name
  namespace: namespace
  finalizers:
  - abc`,

			ExpectedObject: `metadata:
  finalizers:
  - a012
  - def
  name: name
  namespace: namespace`,

			Adjust: func(p Patch) {
				p.Set("metadata.finalizers[0]", "a012")
				finalizers, _ := p.Value("metadata.finalizers")
				arrFinalizers := finalizers.([]interface{})
				arrFinalizers = append(arrFinalizers, "def")
				p.Set("metadata.finalizers", arrFinalizers)
			},
		},
		{
			Name: "Set 2",

			Object: `metadata:
  name: name
  namespace: namespace
  finalizers:
  - abc`,

			ExpectedObject: `metadata:
  finalizers:
  - a012
  name: name
  namespace: namespace`,

			Adjust: func(p Patch) {
				p.Set("metadata.finalizers[0]", "a012")
			},
		},
		{
			Name: "Set 3",

			Object: `metadata:
  finalizers:
  - abc
  - def
  - yyy`,

			ExpectedObject: `metadata:
  finalizers:
  - aaa
  - aaa
  - aaa`,

			Adjust: func(p Patch) {
				p.Set("metadata.finalizers[*]", "aaa")
			},
		},
		{
			Name: "Set 4",

			Object: `
spec:
  selector:
    app: istio-ingress
  servers:
    - hosts:
      - foo01/foo.ingress.loft.sh
      - foo02/catalyst2-admin.ingress.loft.sh
      port:
        name: http
        number: 80
        protocol: HTTP
      tls:
        httpsRedirect: true
    - hosts:
      - foo11/foo.ingress.loft.sh
      port:
        name: https-443
        number: 443
        protocol: HTTPS
      tls:
        credentialName: foo-ui-tls-cert
        mode: SIMPLE
`,

			ExpectedObject: `
spec:
  selector:
    app: istio-ingress
  servers:
  - hosts:
    - aaa
    - aaa
    port:
      name: http
      number: 80
      protocol: HTTP
    tls:
      httpsRedirect: true
  - hosts:
    - aaa
    port:
      name: https-443
      number: 443
      protocol: HTTPS
    tls:
      credentialName: foo-ui-tls-cert
      mode: SIMPLE

`,

			Adjust: func(p Patch) {
				p.Set("spec.servers[*].hosts[*]", "aaa")
			},
		},
		{
			Name: "Translate 1",

			Object: `spec:
  rules:
  - host: foo.bar.com
  - host: "*.foo.com"`,

			ExpectedObject: `spec:
  rules:
  - host: vcluster.foo.bar.com
  - host: vcluster.*.foo.com`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.rules[*].host", func(_ string, val interface{}) (interface{}, error) {
					return "vcluster." + val.(string), nil
				})
			},
		},
		{
			Name: "Translate 2",

			Object: `spec:
  rules:
    abc:
      host: foo.bar.com
    def:
      host: "*.foo.com"`,

			ExpectedObject: `spec:
  rules:
    abc:
      host: vcluster.foo.bar.com
    def:
      host: vcluster.*.foo.com`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.rules.*.host", func(_ string, val interface{}) (interface{}, error) {
					return "vcluster." + val.(string), nil
				})
			},
		},
		{
			Name: "Has",

			Object: `spec:
  test: test
  rules:
  - other
  - host:
      other: other`,

			ExpectedObject: `spec:
  hostname: test
  rules:
  - other
  - host:
      other: other`,

			Adjust: func(p Patch) {
				if !p.Has("spec.test") {
					panic("expected spec.test")
				}
				if !p.Has("spec.rules[0]") {
					panic("expected spec.rules[0]")
				}
				if !p.Has("spec.rules[1].host.other") {
					panic("expected spec.rules[1].host.other")
				}
				if !p.Has("spec.rules[*].host.other") {
					panic("expected spec.rules[*].host.other")
				}
				if p.Has("spec.rules[*].other") {
					panic("unexpected spec.rules[*].other")
				}
				p.Set("spec.hostname", "test")
				p.Delete("spec.test")
				p.DeleteAllExcept("", "spec")
			},
		},
		{
			Name: "Path",

			Object: `spec:
  rules:
   test.object.other:
   - hello
   - world`,

			ExpectedObject: `spec:
  rules:
    test.object.other:
    - spec.rules."test.object.other".0
    - spec.rules."test.object.other".1
    - added`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.rules[*][*]", func(path string, _ interface{}) (interface{}, error) {
					return path, nil
				})
				p.MustTranslate("spec.rules[\"test.object.other\"]", func(_ string, val interface{}) (interface{}, error) {
					valArr := val.([]interface{})
					valArr = append(valArr, "added")
					return valArr, nil
				})
				if val, ok := p.String("spec.rules.\"test.object.other\".1"); !ok || val != "spec.rules.\"test.object.other\".1" {
					panic("Unexpected value at spec.rules.\"test.object.other\".1: " + val)
				}
			},
		},
		{
			Name: "Path multiple wildcards",

			Object: `spec:
  selector:
    app: istio-ingress
  servers:
    - hosts:
      - foo01/foo.ingress.loft.sh
      - foo02/catalyst2-admin.ingress.loft.sh
      port:
        name: http
        number: 80
        protocol: HTTP
      tls:
        httpsRedirect: true
    - hosts:
      - foo11/foo.ingress.loft.sh
      port:
        name: https-443
        number: 443
        protocol: HTTPS
      tls:
        credentialName: foo-ui-tls-cert
        mode: SIMPLE`,

			ExpectedObject: `
spec:
  selector:
    app: istio-ingress
  servers:
  - hosts:
    - ./foo.ingress.loft.sh
    - ./catalyst2-admin.ingress.loft.sh
    port:
      name: http
      number: 80
      protocol: HTTP
    tls:
      httpsRedirect: true
  - hosts:
    - ./foo.ingress.loft.sh
    port:
      name: https-443
      number: 443
      protocol: HTTPS
    tls:
      credentialName: foo-ui-tls-cert
      mode: SIMPLE
`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.servers[*].hosts[*]", func(_ string, val interface{}) (interface{}, error) {
					valString, ok := val.(string)
					if !ok {
						return val, nil
					}
					_, dnsName, found := strings.Cut(valString, "/")
					if !found {
						return val, nil
					}

					return "./" + dnsName, nil
				})
			},
		},
		{
			Name: "Handle nil",

			Object: `spec:
  rules: null`,

			ExpectedObject: `spec:
  rules: test`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.rules", func(_ string, val interface{}) (interface{}, error) {
					_, ok := val.(string)
					if !ok {
						return "test", nil
					}

					return val, nil
				})
			},
		},
		{
			Name: "Set advanced",

			Object: `spec:
  other123: null
  containers:
  - abc: {}
  - def: {}
  - hij: {}
  rules: null`,

			ExpectedObject: `spec:
  containers:
  - abc: {}
    test:
      test: test1234
  - def: {}
    test:
      test: test1234
  - hij: {}
    test:
      test: test1234
  other:
    test:
    - null
    - test1234
    - other: test1234
    - test123
  other123:
    test: test1234
  rules:
  - null
  - test`,

			Adjust: func(p Patch) {
				p.Set("spec.rules", []interface{}{})
				p.Set("spec.rules[1]", "test")
				p.Set("spec.other.test[3]", "test123")
				p.Set("spec.other.test[1]", "test1234")
				p.Set("spec.containers[*].test.test", "test1234")
				p.Set("spec.other123.test", "test1234")
				p.Set("spec.other.test[2].other", "test1234")
			},
		},
	}

	for _, singleTest := range tests {
		t.Run(singleTest.Name, func(t *testing.T) {
			p := Patch{}
			err := yaml.Unmarshal([]byte(singleTest.Object), &p)
			assert.NilError(t, err)

			singleTest.Adjust(p)

			bytes, err := yaml.Marshal(p)
			assert.NilError(t, err)
			assert.Equal(t, strings.TrimSpace(singleTest.ExpectedObject), strings.TrimSpace(string(bytes)))
		})
	}
}
