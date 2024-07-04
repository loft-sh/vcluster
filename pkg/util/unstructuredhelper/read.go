package unstructuredhelper

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadFrom(obj client.Object) ReadMap {
	return obj.(*unstructured.Unstructured).Object
}

type ReadArray []interface{}

func (a ReadArray) Exists() bool {
	return a != nil
}

func (a ReadArray) String(index int) string {
	if index >= len(a) {
		return ""
	}

	retObj, ok := a[index].(string)
	if !ok {
		return ""
	}

	return retObj
}

func (a ReadArray) Bool(index int) bool {
	if index >= len(a) {
		return false
	}

	retObj, ok := a[index].(bool)
	if !ok {
		return false
	}

	return retObj
}

func (a ReadArray) Map(index int) ReadMap {
	if index >= len(a) {
		return nil
	}

	retObj, ok := a[index].(map[string]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (a ReadArray) Array(index int) ReadArray {
	if index >= len(a) {
		return nil
	}

	retObj, ok := a[index].([]interface{})
	if !ok {
		return nil
	}

	return retObj
}

type ReadMap map[string]interface{}

func (m ReadMap) Exists() bool {
	return m != nil
}

func (m ReadMap) Has(key string) bool {
	if m == nil {
		return false
	}

	_, ok := m[key]
	return ok
}

func (m ReadMap) Bool(key string) bool {
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

func (m ReadMap) Array(key string) ReadArray {
	if m == nil {
		return nil
	}

	obj, ok := m[key]
	if !ok {
		return ReadArray{}
	}

	retObj, ok := obj.([]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (m ReadMap) Map(key string) ReadMap {
	if m == nil {
		return nil
	}

	obj, ok := m[key]
	if !ok {
		return ReadMap{}
	}

	retObj, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}

	return retObj
}

func (m ReadMap) String(key string) string {
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
