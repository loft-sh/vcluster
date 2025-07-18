package unstructured

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ToPartialUnstructured converts an object to unstructured, but only converts specific field paths
// This is more memory efficient than using runtime.DefaultUnstructuredConverter since that requires the full
// object to be converted and stored before extracting specific values from that object
func ToPartialUnstructured(obj interface{}, fieldPaths ...string) map[string]interface{} {
	if u, ok := obj.(unstructured.Unstructured); ok {
		obj = u.UnstructuredContent()
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		obj = u.UnstructuredContent()
	}

	result := make(map[string]interface{})
	for _, fieldPath := range fieldPaths {
		_ = extractNestedField(obj, result, lo.Filter(strings.Split(fieldPath, "."), func(s string, _ int) bool { return s != "" })...)
	}
	return result
}

// extractNestedField extracts a field using a path and populates the result map accordingly
func extractNestedField(obj interface{}, result map[string]interface{}, field ...string) error {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var val reflect.Value
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.Type().NumField() {
			f := v.Type().Field(i)
			tag := getJSONKey(f)
			if f.Name == field[0] || tag == field[0] {
				val = v.Field(i)
				break
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if key.String() == field[0] {
				val = v.MapIndex(key)
				break
			}
		}
	default:
	}
	if !val.IsValid() {
		return fmt.Errorf("field %q not found in %T", field[0], obj)
	}
	if len(field) == 1 {
		// Final field — assign directly
		result[field[0]] = val.Interface()
		return nil
	}
	// Intermediate map — recurse
	childMap := map[string]interface{}{}
	err := extractNestedField(val.Interface(), childMap, field[1:]...)
	if err != nil {
		return err
	}
	// Merge into parent map
	if _, ok := result[field[0]]; !ok {
		result[field[0]] = map[string]interface{}{}
	}
	for k, v := range childMap {
		m, ok := result[field[0]].(map[string]interface{})
		// In general, this should never happen because we have a check higher up in the function for field existence
		if !ok {
			panic(fmt.Sprintf("full field path %q not found in %T", field, obj))
		}
		m[k] = v
	}
	return nil
}

// getJSONKey returns the JSON key from a struct tag
func getJSONKey(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
		return tag[:commaIdx]
	}
	return tag
}
