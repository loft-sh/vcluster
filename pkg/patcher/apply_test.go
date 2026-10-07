package patcher

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSanitizePatchForLog(t *testing.T) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "ns"}}
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}}

	tests := []struct {
		name  string
		obj   client.Object
		patch string
		// check receives the sanitized output and calls t.Fatal on unexpected results.
		check func(t *testing.T, got string)
	}{
		{
			name:  "non-Secret returns patch verbatim",
			obj:   configMap,
			patch: `{"data":{"key":"value"}}`,
			check: wantExact(`{"data":{"key":"value"}}`),
		},
		{
			name:  "Secret data values are redacted, keys preserved",
			obj:   secret,
			patch: `{"data":{"username":"c2VjcmV0LW5ldw==","password":"c2VjcmV0"}}`,
			check: wantDataValues(map[string]string{
				"username": "[REDACTED]",
				"password": "[REDACTED]",
			}),
		},
		{
			name:  "Secret stringData values are redacted, keys preserved",
			obj:   secret,
			patch: `{"stringData":{"api-key":"plaintext-secret"}}`,
			check: wantStringDataValues(map[string]string{
				"api-key": "[REDACTED]",
			}),
		},
		{
			name:  "Secret with both data and stringData are both redacted",
			obj:   secret,
			patch: `{"data":{"token":"dG9rZW4="},"stringData":{"extra":"plain"}}`,
			check: func(t *testing.T, got string) {
				t.Helper()
				wantDataValues(map[string]string{"token": "[REDACTED]"})(t, got)
				wantStringDataValues(map[string]string{"extra": "[REDACTED]"})(t, got)
			},
		},
		{
			name:  "Secret patch with no data or stringData is returned as-is",
			obj:   secret,
			patch: `{"metadata":{"labels":{"app":"test"}}}`,
			check: wantExact(`{"metadata":{"labels":{"app":"test"}}}`),
		},
		{
			name:  "Secret data null is preserved verbatim (no keys to redact)",
			obj:   secret,
			patch: `{"data":null}`,
			check: wantExact(`{"data":null}`),
		},
		{
			name:  "malformed JSON returns [REDACTED] sentinel",
			obj:   secret,
			patch: `not-valid-json`,
			check: wantExact("[REDACTED]"),
		},
		{
			// kubectl apply stores the full manifest – including secret data –
			// as a JSON string in this annotation. The manifest is parsed as a
			// corev1.Secret and its data / stringData values are replaced with
			// "[REDACTED]" while keys and all other fields (kind, metadata…)
			// are preserved.
			name: "last-applied-configuration annotation: manifest data values are redacted, keys and structure preserved",
			obj:  secret,
			patch: `{"data":{"username":"[REDACTED]"},` +
				`"metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":` +
				`"{\"apiVersion\":\"v1\",\"data\":{\"password\":\"c2VjcmV0\",\"username\":\"YWRtaW4=\"},` +
				`\"kind\":\"Secret\",\"metadata\":{\"name\":\"my-secret\",\"namespace\":\"default\"},` +
				`\"type\":\"Opaque\"}"}}}`,
			check: func(t *testing.T, got string) {
				t.Helper()

				// Parse the sanitized output as a generic map.
				var top map[string]json.RawMessage
				if err := json.Unmarshal([]byte(got), &top); err != nil {
					t.Fatalf("output is not valid JSON: %v\ngot: %s", err, got)
				}

				// Navigate metadata → annotations → annotation key.
				var meta map[string]json.RawMessage
				if err := json.Unmarshal(top["metadata"], &meta); err != nil {
					t.Fatalf("cannot parse metadata: %v\ngot: %s", err, got)
				}
				var annotations map[string]json.RawMessage
				if err := json.Unmarshal(meta["annotations"], &annotations); err != nil {
					t.Fatalf("cannot parse annotations: %v\ngot: %s", err, got)
				}
				rawAnnotation, ok := annotations[lastAppliedConfigAnnotation]
				if !ok {
					t.Fatalf("annotation key %q missing; got: %s", lastAppliedConfigAnnotation, got)
				}

				// The annotation value is a JSON string containing the sanitised manifest.
				var manifestJSON string
				if err := json.Unmarshal(rawAnnotation, &manifestJSON); err != nil {
					t.Fatalf("annotation value is not a JSON string: %v\ngot: %s", err, got)
				}

				// Parse the embedded manifest.
				var manifest map[string]json.RawMessage
				if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
					t.Fatalf("embedded manifest is not valid JSON: %v\nmanifest: %s", err, manifestJSON)
				}

				// Non-sensitive fields must be preserved.
				assertManifestStringField(t, manifest, "kind", "Secret", manifestJSON)
				assertManifestStringField(t, manifest, "apiVersion", "v1", manifestJSON)

				// data values must be "[REDACTED]", keys must be present.
				var dataMap map[string]string
				if err := json.Unmarshal(manifest["data"], &dataMap); err != nil {
					t.Fatalf("cannot parse manifest.data: %v\nmanifest: %s", err, manifestJSON)
				}
				for _, key := range []string{"password", "username"} {
					val, ok := dataMap[key]
					if !ok {
						t.Fatalf("manifest.data key %q missing; manifest: %s", key, manifestJSON)
					}
					if val != "[REDACTED]" {
						t.Fatalf("manifest.data[%q] = %q, want \"[REDACTED]\"; manifest: %s", key, val, manifestJSON)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizePatchForLog(tc.obj, []byte(tc.patch))
			tc.check(t, got)
		})
	}
}

// wantExact returns a check function that asserts the sanitized output equals s exactly.
func wantExact(s string) func(t *testing.T, got string) {
	return func(t *testing.T, got string) {
		t.Helper()
		if got != s {
			t.Fatalf("sanitizePatchForLog() = %q, want %q", got, s)
		}
	}
}

// wantDataValues returns a check function that asserts each key in want appears
// under the top-level "data" field with the expected value.
func wantDataValues(want map[string]string) func(t *testing.T, got string) {
	return wantFieldValues("data", want)
}

// wantStringDataValues returns a check function that asserts each key in want
// appears under the top-level "stringData" field with the expected value.
func wantStringDataValues(want map[string]string) func(t *testing.T, got string) {
	return wantFieldValues("stringData", want)
}

// assertManifestStringField asserts that the given top-level field in manifest
// is a JSON string with the expected value, and calls t.Fatal otherwise.
func assertManifestStringField(t *testing.T, manifest map[string]json.RawMessage, field, want, manifestJSON string) {
	t.Helper()
	raw, ok := manifest[field]
	if !ok {
		t.Fatalf("manifest field %q missing; manifest: %s", field, manifestJSON)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("manifest field %q is not a string: %v; manifest: %s", field, err, manifestJSON)
	}
	if got != want {
		t.Fatalf("manifest[%q] = %q, want %q; manifest: %s", field, got, want, manifestJSON)
	}
}

// wantFieldValues returns a check function that parses the sanitized output as
// JSON and asserts that each key in want is present under the given top-level
// field with the corresponding value.
func wantFieldValues(field string, want map[string]string) func(t *testing.T, got string) {
	return func(t *testing.T, got string) {
		t.Helper()

		var top map[string]map[string]string
		if err := json.Unmarshal([]byte(got), &top); err != nil {
			t.Fatalf("output is not valid JSON: %v\ngot: %s", err, got)
		}
		fieldMap, ok := top[field]
		if !ok {
			t.Fatalf("field %q missing from output; got: %s", field, got)
		}
		for k, wantVal := range want {
			gotVal, ok := fieldMap[k]
			if !ok {
				t.Fatalf("key %q missing from field %q; got: %s", k, field, got)
			}
			if gotVal != wantVal {
				t.Fatalf("[%q][%q] = %q, want %q", field, k, gotVal, wantVal)
			}
		}
	}
}

type conflictOnceClient struct {
	client.Client

	firstUpdate bool
}

func (c *conflictOnceClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if !c.firstUpdate {
		c.firstUpdate = true

		current := &corev1.ConfigMap{}
		if err := c.Client.Get(ctx, client.ObjectKeyFromObject(obj), current); err != nil {
			return err
		}
		current.Labels = map[string]string{"external": "true"}
		if err := c.Client.Update(ctx, current, opts...); err != nil {
			return err
		}

		return kerrors.NewConflict(schema.GroupResource{Group: "", Resource: "configmaps"}, obj.GetName(), fmt.Errorf("simulated conflict"))
	}

	return c.Client.Update(ctx, obj, opts...)
}

func TestApplyObjectRetriesConflictWithLatestObject(t *testing.T) {
	t.Helper()

	virtualClient := testingutil.NewFakeClient(scheme.Scheme, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
			Labels: map[string]string{
				"initial": "true",
			},
		},
	})

	syncCtx := &synccontext.SyncContext{
		Context:       context.Background(),
		VirtualClient: &conflictOnceClient{Client: virtualClient},
	}

	before := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
			Labels: map[string]string{
				"initial": "true",
			},
		},
	}
	after := before.DeepCopy()
	after.Labels["synced"] = "true"

	if err := ApplyObject(syncCtx, before, after, synccontext.SyncHostToVirtual, false); err != nil {
		t.Fatalf("ApplyObject() error = %v", err)
	}

	updated := &corev1.ConfigMap{}
	if err := virtualClient.Get(context.Background(), client.ObjectKey{Name: "example", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if updated.Labels["synced"] != "true" {
		t.Fatalf("expected synced label to be preserved after retry, got labels: %#v", updated.Labels)
	}
	if updated.Labels["external"] != "true" {
		t.Fatalf("expected external label from concurrent update to survive retry, got labels: %#v", updated.Labels)
	}
}
