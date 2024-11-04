package patcher

import (
	"encoding/json"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/generics"

	jsonpatch "github.com/evanphx/json-patch/v5"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func CopyBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T) {
	newVirtual := virtual
	newHost := host
	if generics.IsNilOrEmpty(host) || !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		newHost = virtual
	} else if generics.IsNilOrEmpty(virtual) || !apiequality.Semantic.DeepEqual(hostOld, host) {
		newVirtual = host
	}

	return newVirtual, newHost
}

func MergeBidirectional[T any](virtualOld, virtual, hostOld, host T) (T, T, error) {
	var err error

	newVirtual := virtual
	newHost := host
	if generics.IsNilOrEmpty(host) || !apiequality.Semantic.DeepEqual(virtualOld, virtual) {
		newHost, err = MergeChangesInto(virtualOld, virtual, host)
	} else if generics.IsNilOrEmpty(virtual) || !apiequality.Semantic.DeepEqual(hostOld, host) {
		newVirtual, err = MergeChangesInto(hostOld, host, virtual)
	}

	return newVirtual, newHost, err
}

func MergeChangesInto[T any](oldValue, newValue, outValue T) (T, error) {
	if generics.IsNilOrEmpty(outValue) {
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
