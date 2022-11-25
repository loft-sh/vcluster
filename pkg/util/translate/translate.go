package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespaceLabel  = "vcluster.loft.sh/namespace"
	MarkerLabel     = "vcluster.loft.sh/managed-by"
	LabelPrefix     = "vcluster.loft.sh/label"
	ControllerLabel = "vcluster.loft.sh/controlled-by"
	Suffix          = "suffix"

	ManagedAnnotationsAnnotation = "vcluster.loft.sh/managed-annotations"
	ManagedLabelsAnnotation      = "vcluster.loft.sh/managed-labels"
)

var Owner client.Object

func SafeConcatGenerateName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 53 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:42]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-")
	}
	return fullPath
}

func SafeConcatName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 63 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:52]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-")
	}
	return fullPath
}

func UniqueSlice(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if entry == "" {
			continue
		}
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func Split(s, sep string) (string, string) {
	parts := strings.SplitN(s, sep, 2)
	return strings.TrimSpace(parts[0]), strings.TrimSpace(safeIndex(parts, 1))
}

func safeIndex(parts []string, idx int) string {
	if len(parts) <= idx {
		return ""
	}
	return parts[idx]
}

func Exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
}
