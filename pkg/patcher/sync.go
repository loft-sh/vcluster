package patcher

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// CopyBidirectional determines whether the change is in the virtual or host object by seeing what changed between "old" and "new" for each.
// It then mutates the changed object to match the unchanged object.
func CopyBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T) {
	newVirtual := virtual
	newHost := host
	if !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		newHost = virtual
	} else if !apiequality.Semantic.DeepEqual(hostOld, host) {
		newVirtual = host
	}

	return newVirtual, newHost
}

// MergeBidirectional determines whether the change is in the virtual or host object by seeing what changed between "old" and "new" for each.
// It then merges the changes from the changed object into the unchanged object by marshalling them and using a json merge patch on the unchanged object.
func MergeBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T, error) {
	var err error

	newVirtual := virtual
	newHost := host
	if !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		// virtual object changed, merge changes into host
		newHost, err = MergeChangesInto(virtualOld, virtual, host)
	} else if !apiequality.Semantic.DeepEqual(hostOld, host) {
		// host object changed, merge changes into virtual
		newVirtual, err = MergeChangesInto(hostOld, host, virtual)
	}

	return newVirtual, newHost, err
}

// MergeChangesInto merges changes from "newValue" into "outValue" based on the changes between "oldValue" and "newValue".
func MergeChangesInto[T any](oldValue, newValue, outValue T) (T, error) {
	if clienthelper.IsNilObject(outValue) {
		return newValue, nil
	}

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
