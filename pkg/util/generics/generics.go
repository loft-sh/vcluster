package generics

import "reflect"

func IsNilOrEmpty[T any](v T) bool {
	i := interface{}(v)

	// Check if the value is nil
	if i == nil {
		return true
	}

	rv := reflect.ValueOf(i)

	// Check if it's nil for types that can be nil
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		if rv.IsNil() {
			return true
		}
	default:
	}

	// Check if it's a zero value
	zeroValue := reflect.Zero(rv.Type()).Interface()
	return reflect.DeepEqual(i, zeroValue)
}

func Filter[T any](slice []T, predicate func(T) bool) []T {
	var filtered []T
	for _, item := range slice {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func Every[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if !predicate(item) {
			return false
		}
	}
	return true
}
