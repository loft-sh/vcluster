package translate

import (
	"crypto/sha256"
	"encoding/hex"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDefaultClusterTranslator(physicalNamespace string, nameTranslator PhysicalNameTranslator, excludedAnnotations ...string) Translator {
	return &defaultClusterTranslator{
		physicalNamespace:   physicalNamespace,
		nameTranslator:      nameTranslator,
		excludedAnnotations: excludedAnnotations,
	}
}

type defaultClusterTranslator struct {
	physicalNamespace   string
	nameTranslator      PhysicalNameTranslator
	excludedAnnotations []string
}

func (d *defaultClusterTranslator) Translate(vObj client.Object) (runtime.Object, error) {
	pObj, err := setupMetadataWithName(d.physicalNamespace, vObj, d.nameTranslator)
	if err != nil {
		return nil, err
	}

	pObj.SetLabels(d.TranslateLabels(vObj))
	pObj.SetAnnotations(d.TranslateAnnotations(vObj, nil))
	return pObj, nil
}

func (d *defaultClusterTranslator) TranslateLabels(vObj client.Object) map[string]string {
	newLabels := map[string]string{}
	if vObj != nil {
		for k, v := range vObj.GetLabels() {
			newLabels[convertNamespacedLabelKey(d.physicalNamespace, k)] = v
		}
	}
	newLabels[MarkerLabel] = SafeConcatName(d.physicalNamespace, "x", Suffix)
	return newLabels
}

func (d *defaultClusterTranslator) TranslateAnnotations(vObj client.Object, pObj client.Object) map[string]string {
	return translateAnnotations(vObj, pObj, d.excludedAnnotations)
}

func convertNamespacedLabelKey(physicalNamespace, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(LabelPrefix, physicalNamespace, "x", Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}
