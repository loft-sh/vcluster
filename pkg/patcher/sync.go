package patcher

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func CopyBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T) {
	newVirtual := virtual
	newHost := host
	if IsNil(virtual) || !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		newHost = virtual
	} else if IsNil(host) || !apiequality.Semantic.DeepEqual(hostOld, host) {
		newVirtual = host
	}

	return newVirtual, newHost
}

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

func CopyBidirectionalFields[T client.Object](event *synccontext.SyncEvent[T], fieldNames ...string) error {
	for _, fieldName := range fieldNames {
		if err := copyBidirectionalField(event, fieldName); err != nil {
			return fmt.Errorf("failed to copy field %s: %w", fieldName, err)
		}
	}
	return nil
}

func copyBidirectionalField[T client.Object](event *synccontext.SyncEvent[T], fieldPath string) error {
	// Get reflect.Value of the event struct
	eventValue := reflect.ValueOf(event).Elem()

	// Get the Virtual and Host objects
	virtualValue := eventValue.FieldByName("Virtual").Elem()
	virtualOldValue := eventValue.FieldByName("VirtualOld").Elem()
	hostValue := eventValue.FieldByName("Host").Elem()
	hostOldValue := eventValue.FieldByName("HostOld").Elem()

	// Get the nested fields
	virtualField, err := getNestedField(virtualValue, fieldPath)
	if err != nil {
		return fmt.Errorf("error accessing Virtual.%s: %w", fieldPath, err)
	}

	virtualOldField, err := getNestedField(virtualOldValue, fieldPath)
	if err != nil {
		return fmt.Errorf("error accessing VirtualOld.%s: %w", fieldPath, err)
	}

	hostField, err := getNestedField(hostValue, fieldPath)
	if err != nil {
		return fmt.Errorf("error accessing Host.%s: %w", fieldPath, err)
	}

	hostOldField, err := getNestedField(hostOldValue, fieldPath)
	if err != nil {
		return fmt.Errorf("error accessing HostOld.%s: %w", fieldPath, err)
	}

	// Get interface values for comparison
	virtualOld := virtualOldField.Interface()
	virtual := virtualField.Interface()
	hostOld := hostOldField.Interface()
	host := hostField.Interface()

	// Apply the bidirectional logic
	if IsNil(virtual) || !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		hostField.Set(virtualField)
	} else if IsNil(host) || !apiequality.Semantic.DeepEqual(hostOld, host) {
		virtualField.Set(hostField)
	}

	return nil
}

func getNestedField(v reflect.Value, fieldPath string) (reflect.Value, error) {
	fields := strings.Split(fieldPath, ".")
	current := v

	for _, field := range fields {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}, fmt.Errorf("nil pointer encountered while accessing %s", field)
			}
			current = current.Elem()
		}

		current = current.FieldByName(field)
		if !current.IsValid() {
			return reflect.Value{}, fmt.Errorf("field %s not found in path %s", field, fieldPath)
		}
	}

	return current, nil
}

func MergeBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T, error) {
	var err error

	newVirtual := virtual
	newHost := host
	if !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		newHost, err = MergeChangesInto(virtualOld, virtual, host)
	} else if !apiequality.Semantic.DeepEqual(hostOld, host) {
		newVirtual, err = MergeChangesInto(hostOld, host, virtual)
	}

	return newVirtual, newHost, err
}

func MergeChangesInto[T any](oldValue, newValue, outValue T) (T, error) {
	var ret T
	oldValueBytes, err := json.Marshal(oldValue)
	if err != nil {
		return ret, fmt.Errorf("marshal old value: %w", err)
	}

	newValueBytes, err := json.Marshal(newValue)
	if err != nil {
		return ret, fmt.Errorf("marshal new value: %w", err)
	}

	outBytes, err := json.Marshal(outValue)
	if err != nil {
		return ret, fmt.Errorf("marshal out value: %w", err)
	}
	if string(outBytes) == "null" {
		outBytes = []byte("{}")
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldValueBytes, newValueBytes)
	if err != nil {
		return ret, fmt.Errorf("create merge patch: %w", err)
	}

	mergedBytes, err := jsonpatch.MergePatch(outBytes, patchBytes)
	if err != nil {
		return ret, fmt.Errorf("merge patch: %w", err)
	}

	err = json.Unmarshal(mergedBytes, &ret)
	if err != nil {
		return ret, fmt.Errorf("unmarshal merged: %w", err)
	}

	return ret, nil
}
