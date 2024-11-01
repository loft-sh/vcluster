package generics

import "reflect"

func IsNil[T any](v T) bool {
	i := interface{}(v)

	if i == nil {
		return true
	}

	rv := reflect.ValueOf(i)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}
