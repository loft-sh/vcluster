package telemetry

import (
	"maps"
	"reflect"
	"strings"

	"github.com/loft-sh/vcluster/config"
)

// sentConfigTypes lists the string types of the config whose value is sent,
// and the check a value has to pass.
var sentConfigTypes = map[reflect.Type]func(string) bool{
	reflect.TypeOf(config.StrBool("")): oneOf("true", "false", "auto"),
	reflect.TypeOf(config.DNSPolicy("")): oneOf(string(config.DNSClusterFirstWithHostNet), string(config.DNSClusterFirst),
		string(config.DNSDefault), string(config.DNSNone)),
}

// sentConfigValues lists, by config path, the plain string fields whose value
// is sent, and the check a value has to pass. A path belongs here only if
// every value its check accepts is safe to send. Paths below a list or a map
// cannot be listed: those are sent as a count.
var sentConfigValues = map[string]func(string) bool{
	"controlPlane.distro.k8s.version":                                  releaseVersion.MatchString,
	"controlPlane.distro.k8s.image.tag":                                releaseVersion.MatchString,
	"controlPlane.statefulSet.scheduling.podManagementPolicy":          oneOf("Parallel", "OrderedReady"),
	"controlPlane.statefulSet.persistence.volumeClaim.retentionPolicy": oneOf("Retain", "Delete"),
	"policies.podSecurityStandard":                                     oneOf("privileged", "baseline", "restricted"),
	"logging.encoding":                                                 oneOf("console", "json"),
}

// configSummary describes what is configured without any free-form value. It
// follows the config types, so the result has the keys of vcluster.yaml and
// nothing a user wrote:
//   - booleans and numbers are sent as they are
//   - a string is sent as true, meaning it is set, unless its type is in
//     sentConfigTypes or its path in sentConfigValues and the value passes
//     the check
//   - lists and maps are sent as their number of entries, so the keys of a
//     map are never sent
//
// Values that are empty, false or zero are left out, except behind a pointer
// or in an untyped field, where false and zero were set on purpose.
func configSummary(vConfig *config.Config) map[string]interface{} {
	summary, _ := summarizeConfigValue(reflect.ValueOf(vConfig), "").(map[string]interface{})
	return summary
}

// summarizeConfigValue returns nil for a value that is left out.
func summarizeConfigValue(value reflect.Value, path string) interface{} {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			return nil
		}

		elem := value.Elem()
		switch {
		case elem.Kind() == reflect.Bool:
			return elem.Bool()
		case elem.CanInt():
			return elem.Int()
		case elem.CanUint():
			return elem.Uint()
		case elem.CanFloat():
			return elem.Float()
		case value.Kind() == reflect.Pointer:
			return summarizeConfigValue(elem, path)
		case elem.Kind() == reflect.Map || elem.Kind() == reflect.Slice:
			return nonZero(elem.Len())
		default:
			return true
		}
	case reflect.Struct:
		summary := map[string]interface{}{}
		for i := range value.NumField() {
			field := value.Type().Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if !field.IsExported() || name == "-" {
				continue
			}

			// an embedded struct has its fields at the level of the struct
			// holding it, as in vcluster.yaml
			if field.Anonymous && name == "" {
				embedded, _ := summarizeConfigValue(value.Field(i), path).(map[string]interface{})
				maps.Copy(summary, embedded)
				continue
			}

			if name == "" {
				name = field.Name
			}
			if fieldSummary := summarizeConfigValue(value.Field(i), strings.TrimPrefix(path+"."+name, ".")); fieldSummary != nil {
				summary[name] = fieldSummary
			}
		}
		if len(summary) == 0 {
			return nil
		}
		return summary
	case reflect.Map, reflect.Slice, reflect.Array:
		return nonZero(value.Len())
	case reflect.Bool:
		if !value.Bool() {
			return nil
		}
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return nonZero(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nonZero(value.Uint())
	case reflect.Float32, reflect.Float64:
		return nonZero(value.Float())
	case reflect.String:
		text := value.String()
		if text == "" {
			return nil
		}

		isValid := sentConfigTypes[value.Type()]
		if isValid == nil {
			isValid = sentConfigValues[path]
		}
		isStrBool := value.Type() == reflect.TypeOf(config.StrBool(""))
		if isValid != nil && isValid(text) {
			// config.StrBool.MarshalJSON writes these two as booleans
			if isStrBool && (text == "true" || text == "false") {
				return text == "true"
			}
			return text
		}
		// true would read as enabled, which the control plane only does for
		// "true"
		if isStrBool {
			return "other"
		}
		return true
	default:
		return nil
	}
}

func nonZero[T comparable](number T) interface{} {
	var zero T
	if number == zero {
		return nil
	}
	return number
}
