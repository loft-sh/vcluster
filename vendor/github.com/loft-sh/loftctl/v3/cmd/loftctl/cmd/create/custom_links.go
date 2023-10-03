package create

import (
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

const linksHelpText = `Labeled Links to annotate the object with.
These links will be visible from the UI. When used with update, existing links will be replaced.
E.g. --link 'Prod=http://exampleprod.com,Dev=http://exampledev.com'`

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
