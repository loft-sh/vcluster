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
				p.MustTranslate("metadata.finalizers[0]", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return val.(string) + "-translated", nil
				})
				p.MustTranslate("metadata.name", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("metadata.finalizers[1]", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("metadata.namespace", func(_ string, _ interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("spec.rules[*].host", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("spec.rules.*.host", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("spec.rules[*][*]", func(path string, _ interface{}, _ bool) (interface{}, error) {
					return path, nil
				})
				p.MustTranslate("spec.rules[\"test.object.other\"]", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("spec.servers[*].hosts[*]", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
				p.MustTranslate("spec.rules", func(_ string, val interface{}, _ bool) (interface{}, error) {
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
		{
			Name: "Complex Bracket Notation",

			Object: `metadata:
  annotations:
    app.kubernetes.io/name: test
    app.kubernetes.io/version: "1.0"
  labels:
    environment: prod`,

			ExpectedObject: `metadata:
  annotations:
    app.kubernetes.io/name: test-modified
    app.kubernetes.io/version: "2.0"
  labels:
    environment: prod-updated`,

			Adjust: func(p Patch) {
				p.MustTranslate("metadata.annotations[\"app.kubernetes.io/name\"]", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return val.(string) + "-modified", nil
				})
				p.Set("metadata.annotations[\"app.kubernetes.io/version\"]", "2.0")
				p.MustTranslate("metadata.labels[\"environment\"]", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return val.(string) + "-updated", nil
				})
			},
		},
		{
			Name: "Mixed Notation Access",

			Object: `config:
  servers:
    server.domain.com:
      port: 8080
      ssl: true
  databases:
  - name: primary
    host: db1.domain.com
  - name: secondary
    host: db2.domain.com`,

			ExpectedObject: `config:
  databases:
  - host: new-db1.domain.com
    name: primary
  - host: new-db2.domain.com
    name: secondary
  servers:
    server.domain.com:
      port: 9090
      ssl: true`,

			Adjust: func(p Patch) {
				p.Set("config.servers[\"server.domain.com\"].port", 9090)
				p.MustTranslate("config.databases[*].host", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return "new-" + val.(string), nil
				})
			},
		},
		{
			Name: "Nested Bracket Operations",

			Object: `spec:
  ingress:
    rules:
      api.example.com:
        paths:
        - path: /v1
          backend: service1
        - path: /v2
          backend: service2`,

			ExpectedObject: `spec:
  ingress:
    rules:
      api.example.com:
        paths:
        - backend: updated-service1
          path: /v1
        - backend: updated-service2
          path: /v2`,

			Adjust: func(p Patch) {
				p.MustTranslate("spec.ingress.rules[\"api.example.com\"].paths[*].backend", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return "updated-" + val.(string), nil
				})
			},
		},
		{
			Name: "Special Characters in Keys",

			Object: `data:
  config.yaml: |
    key: value
  secret-123: encrypted
  "my/special@key": special-value`,

			ExpectedObject: `data:
  config.yaml: |
    key: modified-value
  my/special@key: updated-special-value
  secret-123: encrypted-updated`,

			Adjust: func(p Patch) {
				p.MustTranslate("data[\"config.yaml\"]", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return strings.Replace(val.(string), "value", "modified-value", 1), nil
				})
				p.Set("data[\"my/special@key\"]", "updated-special-value")
				p.MustTranslate("data[\"secret-123\"]", func(_ string, val interface{}, _ bool) (interface{}, error) {
					return val.(string) + "-updated", nil
				})
			},
		},
		{
			Name: "Array and Map Wildcard Mix",

			Object: `services:
- name: frontend
  endpoints:
    api.example.com: active
    cdn.example.com: disabled
- name: backend
  endpoints:
    db.example.com: active`,

			ExpectedObject: `services:
- endpoints:
    api.example.com: updated
    cdn.example.com: updated
  name: frontend
- endpoints:
    db.example.com: updated
  name: backend`,

			Adjust: func(p Patch) {
				p.Set("services[*].endpoints[*]", "updated")
			},
		},
		{
			Name: "Has Method with Brackets",

			Object: `metadata:
  annotations:
    cert-manager.io/issuer: letsencrypt
    nginx.ingress.kubernetes.io/rewrite-target: /
  labels:
    app: myapp`,

			ExpectedObject: `metadata:
  annotations:
    cert-manager.io/issuer: letsencrypt
    nginx.ingress.kubernetes.io/rewrite-target: /
  labels:
    app: myapp
    verified: "true"`,

			Adjust: func(p Patch) {
				if !p.Has("metadata.annotations[\"cert-manager.io/issuer\"]") {
					panic("expected cert-manager.io/issuer annotation")
				}
				if !p.Has("metadata.annotations[\"nginx.ingress.kubernetes.io/rewrite-target\"]") {
					panic("expected nginx rewrite annotation")
				}
				if p.Has("metadata.annotations[\"nonexistent.key\"]") {
					panic("unexpected nonexistent key")
				}
				p.Set("metadata.labels[\"verified\"]", "true")
			},
		},
		{
			Name: "Undefined vs Null Values",

			Object: `metadata:
  annotations:
    existing-key: "value"
    null-key: null
  labels:
    app: myapp`,

			ExpectedObject: `metadata:
  annotations:
    existing-key: value-modified
    new-key: created
    null-key: was-null
    undefined-key: was-undefined-value
  labels:
    app: myapp`,

			Adjust: func(p Patch) {
				// Modify existing key
				p.MustTranslate("metadata.annotations[\"existing-key\"]", func(_ string, val interface{}, exists bool) (interface{}, error) {
					if !exists {
						panic("expected existing-key to exist")
					}
					return val.(string) + "-modified", nil
				})

				// Handle null value
				p.MustTranslate("metadata.annotations[\"null-key\"]", func(_ string, val interface{}, exists bool) (interface{}, error) {
					if !exists {
						panic("expected null-key to exist")
					}
					if val == nil {
						return "was-null", nil
					}
					return val, nil
				})

				// Handle undefined/non-existent key
				p.MustTranslate("metadata.annotations[\"undefined-key\"]", func(_ string, _ interface{}, exists bool) (interface{}, error) {
					if exists {
						panic("unexpected undefined-key to exist")
					}
					return "was-undefined-value", nil
				})

				// Set a new key to show creation works
				p.Set("metadata.annotations[\"new-key\"]", "created")

				// Verify Has method behavior
				if !p.Has("metadata.annotations[\"existing-key\"]") {
					panic("expected existing-key")
				}
				if !p.Has("metadata.annotations[\"null-key\"]") {
					panic("expected null-key to exist (even though value is null)")
				}
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
