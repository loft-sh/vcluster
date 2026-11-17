package options

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// PopulateStructFromMap fills in a struct pointed to by ptr using values from m.
// It uses the "url" tag to map keys from m to struct fields.
// The tag can include an option "base64" (e.g. `url:"name,base64"`) to indicate that
// the passed data is base64 encoded and should be decoded.
// When strict is true, an error is returned if m holds a key that is not part of the struct.
func PopulateStructFromMap(ptr interface{}, m map[string][]string, strict bool) error {
	// Ensure ptr is a non-nil pointer to a struct.
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("expected non-nil pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("expected pointer to a struct")
	}
	t := v.Type()

	allowedKeys := make(map[string]bool)
	// Iterate over each field in the struct.
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("url")
		if tag == "" {
			continue // Skip fields without a "url" tag.
		}

		// Parse tag: first part is key name; subsequent parts are options.
		parts := strings.Split(tag, ",")
		key := parts[0]
		allowedKeys[key] = true
		base64Flag := false
		if len(parts) > 1 {
			for _, opt := range parts[1:] {
				if opt == "base64" {
					base64Flag = true
					break
				}
			}
		}

		values, ok := m[key]
		if !ok || len(values) == 0 {
			continue // No corresponding value in the map.
		}
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}

		// If the field is a slice of strings, process each element.
		if f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
			result := make([]string, len(values))
			for idx, val := range values {
				if base64Flag {
					decodedBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.ReplaceAll(val, " ", "+")))
					if err != nil {
						return fmt.Errorf("failed to decode base64 for field %s: %w", fieldType.Name, err)
					}
					result[idx] = strings.TrimSpace(string(decodedBytes))
				} else {
					result[idx] = strings.TrimSpace(val)
				}
			}
			f.Set(reflect.ValueOf(result))
			continue
		}

		// For single value fields, use the first value.
		value := values[0]
		if base64Flag {
			decodedBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.ReplaceAll(value, " ", "+")))
			if err != nil {
				return fmt.Errorf("failed to decode base64 for field %s: %w", fieldType.Name, err)
			}
			value = strings.TrimSpace(string(decodedBytes))
		}

		switch f.Kind() {
		case reflect.String:
			f.SetString(strings.TrimSpace(value))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int for field %s: %w", fieldType.Name, err)
			}
			f.SetInt(iVal)
		case reflect.Float32, reflect.Float64:
			fVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float for field %s: %w", fieldType.Name, err)
			}
			f.SetFloat(fVal)
		case reflect.Bool:
			bVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse bool for field %s: %w", fieldType.Name, err)
			}
			f.SetBool(bVal)
		default:
			return fmt.Errorf("unsupported field type %s for field %s", f.Kind(), fieldType.Name)
		}
	}

	// If strict is enabled, check for keys in m that are not part of the struct.
	if strict {
		for key := range m {
			if !allowedKeys[key] {
				return fmt.Errorf("unknown parameter in url: %s", key)
			}
		}
	}

	return nil
}
