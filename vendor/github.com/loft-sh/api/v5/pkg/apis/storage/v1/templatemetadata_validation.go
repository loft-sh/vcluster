package v1

import (
	"maps"
	"slices"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateTemplateMetadata checks metadata a user asks to be copied onto another object.
// Annotation keys follow the same rules as label keys, and annotation values are free form.
func ValidateTemplateMetadata(fldPath *field.Path, labels, annotations map[string]string) field.ErrorList {
	var errs field.ErrorList

	// Keys are sorted so the reported errors do not come out in map iteration order.
	labelsPath := fldPath.Child("labels")
	for _, key := range slices.Sorted(maps.Keys(labels)) {
		for _, msg := range validation.IsQualifiedName(key) {
			errs = append(errs, field.Invalid(labelsPath, key, msg))
		}
		for _, msg := range validation.IsValidLabelValue(labels[key]) {
			errs = append(errs, field.Invalid(labelsPath.Key(key), labels[key], msg))
		}
	}

	annotationsPath := fldPath.Child("annotations")
	for _, key := range slices.Sorted(maps.Keys(annotations)) {
		for _, msg := range validation.IsQualifiedName(key) {
			errs = append(errs, field.Invalid(annotationsPath, key, msg))
		}
	}

	return errs
}
