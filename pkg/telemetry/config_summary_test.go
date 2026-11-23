package telemetry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"gotest.tools/v3/assert"
)

// fillWithText sets every string, map key and untyped value below value to
// text, with one entry in every map and list, every boolean true and every
// number 1.
func fillWithText(value reflect.Value, text string, depth int) {
	// the config types that contain themselves do so through lists and maps
	if depth > 30 || !value.CanSet() {
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		fillWithText(value.Elem(), text, depth+1)
	case reflect.Interface:
		if value.NumMethod() == 0 {
			value.Set(reflect.ValueOf(map[string]interface{}{text: []interface{}{text}}))
		}
	case reflect.Struct:
		for i := range value.NumField() {
			fillWithText(value.Field(i), text, depth+1)
		}
	case reflect.Map:
		key := reflect.New(value.Type().Key()).Elem()
		fillWithText(key, text, depth+1)
		elem := reflect.New(value.Type().Elem()).Elem()
		fillWithText(elem, text, depth+1)
		value.Set(reflect.MakeMap(value.Type()))
		value.SetMapIndex(key, elem)
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		fillWithText(value.Index(0), text, depth+1)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(1)
	case reflect.String:
		value.SetString(text)
	default:
	}
}

func TestConfigSummaryOfChartConfig(t *testing.T) {
	vConfig := parseChartConfig(t, userValues)

	// the whole config holds the text, so the fixture can show it is not sent
	full, err := json.Marshal(vConfig)
	assert.NilError(t, err)
	assert.Assert(t, strings.Count(string(full), configValue) >= 12, "config holds %d values", strings.Count(string(full), configValue))

	summary := toJSONObject(t, configSummary(&vConfig.Config))
	raw, err := json.Marshal(summary)
	assert.NilError(t, err)
	assert.Assert(t, !strings.Contains(string(raw), configValue), "summary carries a config value: %s", raw)

	expected := map[string]interface{}{
		// from the values
		"controlPlane.backingStore.database.external.enabled":    true,
		"controlPlane.backingStore.database.external.dataSource": true,
		"controlPlane.statefulSet.highAvailability.replicas":     3.0,
		"controlPlane.statefulSet.labels":                        1.0,
		"controlPlane.statefulSet.env":                           1.0,
		"controlPlane.ingress.enabled":                           true,
		"controlPlane.ingress.host":                              true,
		"sync.toHost.ingresses.enabled":                          true,
		"sync.toHost.customResources":                            1.0,
		"sync.fromHost.nodes.enabled":                            true,
		"sync.fromHost.nodes.selector.labels":                    1.0,
		"privateNodes.joinNode.containerd.registry.auth":         1.0,
		"policies.podSecurityStandard":                           "baseline",
		"policies.resourceQuota.enabled":                         true,
		"experimental.deploy.vcluster.manifests":                 true,
		"experimental.deploy.vcluster.helm":                      1.0,
		"telemetry.instanceCreator":                              true,
		// a listed path with a value outside its set
		"logging.encoding": true,
		// from the chart defaults
		"controlPlane.statefulSet.scheduling.podManagementPolicy":          "Parallel",
		"controlPlane.statefulSet.persistence.volumeClaim.retentionPolicy": "Retain",
		"controlPlane.statefulSet.persistence.volumeClaim.enabled":         "auto",
		"sync.toHost.pods.enabled":                                         true,
		"telemetry.enabled":                                                true,
		"controlPlane.statefulSet.enableServiceLinks":                      true,
		// false, empty and zero are left out
		"privateNodes.enabled":                            nil,
		"controlPlane.backingStore.etcd.embedded.enabled": nil,
		"controlPlane.distro.k8s.version":                 nil,
		"experimental.deploy.host":                        nil,
		"sync.toHost.networkPolicies":                     nil,
		"controlPlane.statefulSet.scheduling.affinity":    nil,
	}
	for path, value := range expected {
		assert.DeepEqual(t, lookup(summary, path), value)
	}

	tag, isText := lookup(summary, "controlPlane.distro.k8s.image.tag").(string)
	assert.Assert(t, isText && releaseVersion.MatchString(tag), "kubernetes image tag is %v", lookup(summary, "controlPlane.distro.k8s.image.tag"))
}

func TestConfigSummaryHasNoText(t *testing.T) {
	vConfig := &vclusterconfig.Config{}
	fillWithText(reflect.ValueOf(vConfig).Elem(), configValue, 0)

	// the filled config is one where text is everywhere it can be
	assert.Equal(t, vConfig.ControlPlane.BackingStore.Database.External.DataSource, configValue)
	assert.Equal(t, vConfig.PrivateNodes.JoinNode.Containerd.Registry.Auth[configValue].Password, configValue)
	assert.Equal(t, vConfig.Sync.FromHost.StorageClasses.Enabled, vclusterconfig.StrBool(configValue))

	var assertNoText func(path string, value interface{})
	assertNoText = func(path string, value interface{}) {
		switch typed := value.(type) {
		case map[string]interface{}:
			assert.Assert(t, len(typed) > 0, "%s is empty", path)
			for key, child := range typed {
				assertNoText(path+"."+key, child)
			}
		case bool, float64:
		case string:
			assert.Equal(t, typed, "other", "%s is text", path)
		default:
			t.Errorf("%s is %T %v, expected a boolean, a number or other", path, value, value)
		}
	}

	summary := toJSONObject(t, configSummary(vConfig))
	assertNoText("", summary)

	raw, err := json.Marshal(summary)
	assert.NilError(t, err)
	assert.Assert(t, !strings.Contains(string(raw), configValue), "summary carries a config value: %s", raw)

	// text everywhere is still summarized: set strings, counted maps and lists
	assert.Equal(t, lookup(summary, "controlPlane.backingStore.database.external.dataSource"), true)
	assert.Equal(t, lookup(summary, "privateNodes.joinNode.containerd.registry.auth"), 1.0)
	assert.Equal(t, lookup(summary, "controlPlane.service.spec"), 1.0)
	assert.Equal(t, lookup(summary, "sync.fromHost.storageClasses.enabled"), "other")
}

func TestSentConfigValues(t *testing.T) {
	testCases := map[string]struct {
		// ofType is set for a path that stands for a type in sentConfigTypes
		ofType   interface{}
		accepted []string
		rejected []string
	}{
		"controlPlane.distro.k8s.version":                                  {accepted: []string{"v1.35.10", "1.36.6"}, rejected: []string{"latest", "v1.35", "v1.35.10+" + configValue, "v1.35.10-" + configValue}},
		"controlPlane.distro.k8s.image.tag":                                {accepted: []string{"v1.35.10", "v1.36.0-rc.1"}, rejected: []string{"latest", "v1.35.10-" + configValue, "sha256:" + configValue}},
		"controlPlane.statefulSet.scheduling.podManagementPolicy":          {accepted: []string{"Parallel", "OrderedReady"}, rejected: []string{"parallel"}},
		"controlPlane.statefulSet.persistence.volumeClaim.retentionPolicy": {accepted: []string{"Retain", "Delete"}, rejected: []string{"retain"}},
		"policies.podSecurityStandard":                                     {accepted: []string{"privileged", "baseline", "restricted"}, rejected: []string{"Baseline"}},
		"logging.encoding":                                                 {accepted: []string{"console", "json"}, rejected: []string{"JSON"}},
		"sync.fromHost.storageClasses.enabled":                             {ofType: vclusterconfig.StrBool(""), accepted: []string{"true", "false", "auto"}, rejected: []string{"Auto", "yes"}},
		"controlPlane.statefulSet.dnsPolicy":                               {ofType: vclusterconfig.DNSPolicy(""), accepted: []string{"ClusterFirstWithHostNet", "ClusterFirst", "Default", "None"}, rejected: []string{"default"}},
	}

	// every listed path and type needs a case here, so adding one without
	// saying what it must reject fails
	listed := map[interface{}]bool{}
	for path, testCase := range testCases {
		if testCase.ofType != nil {
			assert.Assert(t, sentConfigTypes[reflect.TypeOf(testCase.ofType)] != nil, "%T is not listed", testCase.ofType)
			listed[reflect.TypeOf(testCase.ofType)] = true
			continue
		}
		assert.Assert(t, sentConfigValues[path] != nil, "%s is not listed", path)
		listed[path] = true
	}
	assert.Equal(t, len(listed), len(sentConfigValues)+len(sentConfigTypes))

	alwaysRejected := []string{configValue, "[]", "a=b", "https://user:" + configValue + "@host", strings.Repeat("1", 64)}
	for path, testCase := range testCases {
		t.Run(path, func(t *testing.T) {
			// set through the parser, so a path that is not a string field of
			// the config fails
			summaryOf := func(value string) interface{} {
				keys := strings.Split(path, ".")
				var nested interface{} = value
				for i := len(keys) - 1; i >= 0; i-- {
					nested = map[string]interface{}{keys[i]: nested}
				}
				raw, err := json.Marshal(nested)
				assert.NilError(t, err)

				vConfig := &vclusterconfig.Config{}
				assert.NilError(t, vConfig.UnmarshalYAMLStrict(raw))

				return lookup(toJSONObject(t, configSummary(vConfig)), path)
			}

			for _, value := range testCase.accepted {
				var expected interface{} = value
				if testCase.ofType == vclusterconfig.StrBool("") && value != "auto" {
					expected = value == "true"
				}
				assert.Equal(t, summaryOf(value), expected)
			}
			var rejectedSummary interface{} = true
			if testCase.ofType == vclusterconfig.StrBool("") {
				rejectedSummary = "other"
			}
			for _, value := range append(testCase.rejected, alwaysRejected...) {
				assert.Equal(t, summaryOf(value), rejectedSummary, "%q should not be sent", value)
			}
		})
	}
}

// textNumber is a number that writes itself to JSON as text.
type textNumber int

func (n textNumber) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s-%d", configValue, int(n))), nil
}

// textStruct writes itself to JSON and prints itself as text.
type textStruct struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (s textStruct) MarshalJSON() ([]byte, error) { return json.Marshal(configValue) }

func (s textStruct) String() string { return configValue }

type EmbeddedFields struct {
	Embedded string `json:"embedded,omitempty"`
}

// fieldKinds has a field for every way the config types hold a value.
type fieldKinds struct {
	EmbeddedFields

	Ignored    string `json:"-"`
	unexported string
	Untagged   string

	Bool   bool    `json:"bool,omitempty"`
	Number int     `json:"number,omitempty"`
	Float  float64 `json:"float,omitempty"`
	Text   string  `json:"text,omitempty"`

	BoolPointer   *bool   `json:"boolPointer,omitempty"`
	NumberPointer *int32  `json:"numberPointer,omitempty"`
	TextPointer   *string `json:"textPointer,omitempty"`

	Untyped interface{} `json:"untyped,omitempty"`

	TextNumber        textNumber  `json:"textNumber,omitempty"`
	TextNumberPointer *textNumber `json:"textNumberPointer,omitempty"`
	TextStruct        textStruct  `json:"textStruct,omitempty"`

	Map  map[string]textStruct `json:"map,omitempty"`
	List []textStruct          `json:"list,omitempty"`
}

func TestSummarizeConfigValue(t *testing.T) {
	type summary = map[string]interface{}
	pointerTo := func(value interface{}) reflect.Value {
		pointer := reflect.New(reflect.TypeOf(value))
		pointer.Elem().Set(reflect.ValueOf(value))
		return pointer
	}
	boolPointer := func(value bool) *bool { return &value }
	numberPointer := func(value int32) *int32 { return &value }
	textPointer := func(value string) *string { return &value }
	textNumberPointer := func(value textNumber) *textNumber { return &value }

	testCases := []struct {
		name     string
		value    fieldKinds
		expected summary
	}{
		{name: "nothing set", value: fieldKinds{}, expected: nil},
		{
			name:     "fields that are not part of the config file",
			value:    fieldKinds{Ignored: configValue, unexported: configValue},
			expected: nil,
		},
		{
			name:     "text",
			value:    fieldKinds{EmbeddedFields: EmbeddedFields{Embedded: configValue}, Untagged: configValue, Text: configValue, TextPointer: textPointer(configValue)},
			expected: summary{"embedded": true, "Untagged": true, "text": true, "textPointer": true},
		},
		{name: "empty text behind a pointer", value: fieldKinds{TextPointer: textPointer("")}, expected: nil},
		{
			name:     "booleans and numbers",
			value:    fieldKinds{Bool: true, Number: 3, Float: 0.5, BoolPointer: boolPointer(true), NumberPointer: numberPointer(7)},
			expected: summary{"bool": true, "number": 3.0, "float": 0.5, "boolPointer": true, "numberPointer": 7.0},
		},
		{
			name:     "false and zero set on purpose",
			value:    fieldKinds{BoolPointer: boolPointer(false), NumberPointer: numberPointer(0)},
			expected: summary{"boolPointer": false, "numberPointer": 0.0},
		},
		{name: "untyped false", value: fieldKinds{Untyped: false}, expected: summary{"untyped": false}},
		{name: "untyped number", value: fieldKinds{Untyped: 2.0}, expected: summary{"untyped": 2.0}},
		{name: "untyped zero", value: fieldKinds{Untyped: int64(0)}, expected: summary{"untyped": 0.0}},
		{name: "untyped text", value: fieldKinds{Untyped: "50%" + configValue}, expected: summary{"untyped": true}},
		{name: "untyped map", value: fieldKinds{Untyped: map[string]interface{}{configValue: 1, "b": configValue}}, expected: summary{"untyped": 2.0}},
		{name: "untyped list", value: fieldKinds{Untyped: []interface{}{configValue}}, expected: summary{"untyped": 1.0}},
		{name: "untyped empty map", value: fieldKinds{Untyped: map[string]interface{}{}}, expected: nil},
		{name: "untyped value that writes itself as text", value: fieldKinds{Untyped: textStruct{Enabled: true}}, expected: summary{"untyped": true}},
		{
			name:     "numbers that write themselves as text",
			value:    fieldKinds{TextNumber: 4, TextNumberPointer: textNumberPointer(5), Untyped: textNumber(6)},
			expected: summary{"textNumber": 4.0, "textNumberPointer": 5.0, "untyped": 6.0},
		},
		{
			name:     "struct that writes itself as text",
			value:    fieldKinds{TextStruct: textStruct{Enabled: true}},
			expected: summary{"textStruct": summary{"enabled": true}},
		},
		{
			name:     "maps and lists of structs",
			value:    fieldKinds{Map: map[string]textStruct{configValue: {Enabled: true}}, List: []textStruct{{Enabled: true}, {}}},
			expected: summary{"map": 1.0, "list": 2.0},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			raw, err := json.Marshal(summarizeConfigValue(pointerTo(testCase.value), ""))
			assert.NilError(t, err)
			assert.Assert(t, !strings.Contains(string(raw), configValue), "summary carries a config value: %s", raw)

			var sent summary
			assert.NilError(t, json.Unmarshal(raw, &sent))
			assert.DeepEqual(t, sent, testCase.expected)
		})
	}
}
