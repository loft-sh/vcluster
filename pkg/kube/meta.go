package kube

import (
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LoftCustomLinksAnnotation is applied to enumerates associated links to external websites
	LoftCustomLinksAnnotation = "loft.sh/custom-links"

	// LoftCustomLinksDelimiter is the separator for the values of the custom links annotation
	LoftCustomLinksDelimiter = "\n"
)

func UpdateLabels(obj metav1.Object, labelList []string) (bool, error) {
	// parse strings to map
	labels, err := parseStringMap(labelList)
	if err != nil {
		return false, fmt.Errorf("cannot parse supplied labels in flag '-l': %w", err)
	}

	objLabels := obj.GetLabels()
	var changed bool
	for key, value := range labels {
		// if the labels are nil, just replace the whole object and exit the loop
		if objLabels == nil {
			changed = true
			objLabels = labels
			break
		}
		existing, found := objLabels[key]
		if !found || (found && existing != value) {
			changed = true
		}
		objLabels[key] = value
	}

	obj.SetLabels(objLabels)
	return changed, nil
}

func UpdateAnnotations(obj metav1.Object, annotationList []string) (bool, error) {
	// parse strings to map
	annotations, err := parseStringMap(annotationList)
	if err != nil {
		return false, fmt.Errorf("cannot parse supplied annotations in flag '-l': %w", err)
	}

	objAnnotations := obj.GetAnnotations()
	var changed bool
	for key, value := range annotations {
		// if the labels are nil, just replace the whole object and exit the loop
		if objAnnotations == nil {
			changed = true
			objAnnotations = annotations
			break
		}
		existing, found := objAnnotations[key]
		if !found || (found && existing != value) {
			changed = true
		}
		objAnnotations[key] = value
	}
	obj.SetAnnotations(objAnnotations)
	return changed, nil
}

// SetCustomLinksAnnotation sets the list of links for the UI to display next to the project member({space/virtualcluster}instance)
// it handles unspecified links (empty) during create and update
func SetCustomLinksAnnotation(obj metav1.Object, links []string) bool {
	var changed bool
	if obj == nil {
		log.GetInstance().Error("SetCustomLinksAnnotation called on nil object")
		return false
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	if len(links) > 0 {
		var trimmedLinks string
		for i, link := range links {
			trimmedLink := strings.TrimSpace(link)
			if trimmedLink != "" {
				if i != 0 {
					trimmedLink = LoftCustomLinksDelimiter + trimmedLink
				}
				trimmedLinks += trimmedLink
			}
		}
		if trimmedLinks != "" {
			changed = true
			annotations[LoftCustomLinksAnnotation] = trimmedLinks
		}
	}
	obj.SetAnnotations(annotations)
	return changed
}

func parseStringMap(entries []string) (map[string]string, error) {
	var errList []error
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		var key, value string
		if entry == "" {
			continue
		}
		splitted := strings.Split(entry, "=")
		if len(splitted) > 2 {
			errList = append(errList, fmt.Errorf("cannot parse label entry: %q", entry))
		}
		if len(splitted) > 0 {
			key = splitted[0]
		}
		if len(splitted) > 1 {
			value = splitted[1]
		}
		result[key] = value
	}
	return result, errors.Join(errList...)
}
