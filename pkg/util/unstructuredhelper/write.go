package unstructuredhelper

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func ValueFrom[T any](path string, from ReadMap) (T, bool) {
	if from == nil {
		var ret T
		return ret, false
	}

	pathSplitted := strings.Split(path, ".")

	// traverse to target
	targetMap := from
	for i := 0; i < len(pathSplitted)-1; i++ {
		targetMap = targetMap.Map(pathSplitted[i])

		if targetMap == nil {
			var ret T
			return ret, false
		}
	}

	// check if map has latest path
	lastKey := pathSplitted[len(pathSplitted)-1]
	if !targetMap.Has(lastKey) {
		var ret T
		return ret, false
	}

	// try to convert target value
	targetValue, ok := targetMap[lastKey].(T)
	if !ok {
		var ret T
		return ret, false
	}

	return targetValue, true
}

type TranslateFn[T any] func(in T) T

func Set[T any](path string, from ReadMap, to WriteMap, translate TranslateFn[T]) {
	if from == nil || to == nil {
		return
	}

	pathSplitted := strings.Split(path, ".")

	// traverse to target
	targetMap := from
	toMap := to
	for i := 0; i < len(pathSplitted)-1; i++ {
		targetMap = targetMap.Map(pathSplitted[i])
		toMap = toMap.Map(pathSplitted[i])

		if targetMap == nil || toMap == nil {
			return
		}
	}

	// check if map has latest path
	lastKey := pathSplitted[len(pathSplitted)-1]

	// try to convert target value
	var targetValue T
	if targetMap.Has(lastKey) {
		var ok bool
		targetValue, ok = targetMap[lastKey].(T)
		if !ok {
			return
		}
	}

	toMap[lastKey] = translate(targetValue)
}

func TranslateArray[T any](path string, from ReadMap, to WriteMap, translate TranslateFn[T]) {
	pathSplitted := strings.Split(path, ".")

	// traverse to target
	targetMap := from
	if targetMap == nil {
		return
	}

	for i := 0; i < len(pathSplitted)-1; i++ {
		targetMap = targetMap.Map(pathSplitted[i])

		if targetMap == nil {
			return
		}
	}

	// check if map has latest path
	lastKey := pathSplitted[len(pathSplitted)-1]
	if !targetMap.Has(lastKey) {
		return
	}

	// to map
	toMap := to
	for i := 0; i < len(pathSplitted)-1; i++ {
		if toMap == nil {
			return
		}

		toMap = toMap.Map(pathSplitted[i])
	}

	// try to get source value
	sourceArray, ok := toMap[lastKey].([]T)
	if !ok {
		return
	}

	for i := range sourceArray {
		sourceArray[i] = translate(sourceArray[i])
	}
}

func Translate[T any](path string, from ReadMap, to WriteMap, translate TranslateFn[T]) {
	if from == nil || to == nil {
		return
	}

	pathSplitted := strings.Split(path, ".")

	// traverse to target
	targetMap := from
	for i := 0; i < len(pathSplitted)-1; i++ {
		targetMap = targetMap.Map(pathSplitted[i])

		if targetMap == nil {
			return
		}
	}

	// check if map has latest path
	lastKey := pathSplitted[len(pathSplitted)-1]
	if !targetMap.Has(lastKey) {
		return
	}

	// to map
	toMap := to
	for i := 0; i < len(pathSplitted)-1; i++ {
		toMap = toMap.Map(pathSplitted[i])

		if toMap == nil {
			return
		}
	}

	// try to convert target value
	targetValue, ok := targetMap[lastKey].(T)
	if !ok {
		return
	}

	toMap[lastKey] = translate(targetValue)
}

func WriteInto(obj client.Object) WriteMap {
	return obj.(*unstructured.Unstructured).Object
}

type WriteArray []interface{}

func (a WriteArray) Exists() bool {
	return a != nil
}

func (a WriteArray) String(index int) string {
	if index >= len(a) {
		return ""
	}

	retObj, ok := a[index].(string)
	if !ok {
		return ""
	}

	return retObj
}

func (a WriteArray) Bool(index int) bool {
	if index >= len(a) {
		return false
	}

	retObj, ok := a[index].(bool)
	if !ok {
		return false
	}

	return retObj
}

func (a WriteArray) Map(index int) WriteMap {
	if index >= len(a) {
		return nil
	}

	retObj, ok := a[index].(map[string]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (a WriteArray) Array(index int) WriteArray {
	if index >= len(a) {
		return nil
	}

	retObj, ok := a[index].([]interface{})
	if !ok {
		return nil
	}

	return retObj
}

type WriteMap map[string]interface{}

func (m WriteMap) Exists() bool {
	return m != nil
}

func (m WriteMap) Has(key string) bool {
	if m == nil {
		return false
	}

	_, ok := m[key]
	return ok
}

func (m WriteMap) Bool(key string) bool {
	if m == nil {
		return false
	}

	obj, ok := m[key]
	if !ok {
		return false
	}

	retObj, ok := obj.(bool)
	if !ok {
		return false
	}

	return retObj
}

func (m WriteMap) Array(key string) WriteArray {
	if m == nil {
		return nil
	}

	obj, ok := m[key]
	if !ok {
		m[key] = []interface{}{}
		return m[key].([]interface{})
	}

	retObj, ok := obj.([]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (m WriteMap) Set(key string, val interface{}) {
	readMap, ok := val.(ReadMap)
	if ok {
		m[key] = map[string]interface{}(readMap)
		return
	}

	writeMap, ok := val.(WriteMap)
	if ok {
		m[key] = map[string]interface{}(writeMap)
		return
	}

	readArray, ok := val.(ReadArray)
	if ok {
		m[key] = []interface{}(readArray)
		return
	}

	writeArray, ok := val.(WriteArray)
	if ok {
		m[key] = []interface{}(writeArray)
		return
	}

	m[key] = val
}

func (m WriteMap) M(key string) WriteMap {
	return m.Map(key)
}

func (m WriteMap) Map(key string) WriteMap {
	if m == nil {
		return nil
	}

	obj, ok := m[key]
	if !ok {
		m[key] = map[string]interface{}{}
		return m[key].(map[string]interface{})
	}

	retObj, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (m WriteMap) String(key string) string {
	if m == nil {
		return ""
	}

	str, ok := m[key]
	if !ok {
		return ""
	}

	strVal, ok := str.(string)
	if !ok {
		return ""
	}

	return strVal
}

func (m WriteMap) ToString() string {
	out, _ := yaml.Marshal(m)
	return string(out)
}
