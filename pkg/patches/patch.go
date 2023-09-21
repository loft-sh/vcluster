package patches

import (
	"encoding/json"
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jsonyaml "github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NameResolver interface {
	TranslateName(name string, regex *regexp.Regexp, path string) (string, error)
	TranslateLabelKey(key string) (string, error)
	TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error)
	TranslateLabelSelector(selector map[string]string) (map[string]string, error)
	TranslateNameWithNamespace(name string, namespace string, regex *regexp.Regexp, path string) (string, error)
	TranslateNamespaceRef(namespace string) (string, error)
}

func ApplyPatches(destObj, sourceObj client.Object, patchesConf []*config.Patch, reversePatchesConf []*config.Patch, nameResolver NameResolver) error {
	node1, err := NewJSONNode(destObj)
	if err != nil {
		return errors.Wrap(err, "new json yaml node")
	}

	var node2 *yaml.Node
	if sourceObj != nil {
		node2, err = NewJSONNode(sourceObj)
		if err != nil {
			return errors.Wrap(err, "new json yaml node")
		}
	}

	for _, p := range patchesConf {
		err := applyPatch(node1, node2, p, nameResolver)
		if err != nil {
			return errors.Wrap(err, "apply patch")
		}
	}

	// remove ignore paths from patched object
	for _, p := range reversePatchesConf {
		if p.Path == "" || (p.Ignore != nil && *p.Ignore) {
			continue
		}

		err := applyPatch(node1, node2, &config.Patch{
			Operation: config.PatchTypeRemove,
			Path:      p.Path,
		}, nameResolver)
		if err != nil {
			return errors.Wrap(err, "apply patch")
		}
	}

	objYaml, err := yaml.Marshal(node1)
	if err != nil {
		return errors.Wrap(err, "marshal yaml")
	}

	err = jsonyaml.Unmarshal(objYaml, destObj)
	if err != nil {
		return errors.Wrap(err, "convert object")
	}

	return nil
}

func applyPatch(obj1, obj2 *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	switch patch.Operation {
	case config.PatchTypeRewriteName:
		return RewriteName(obj1, patch, resolver)
	case config.PatchTypeRewriteLabelKey:
		return RewriteLabelKey(obj1, patch, resolver)
	case config.PatchTypeRewriteLabelExpressionsSelector:
		return RewriteLabelExpressionsSelector(obj1, patch, resolver)
	case config.PatchTypeRewriteLabelSelector:
		return RewriteLabelSelector(obj1, patch, resolver)
	case config.PatchTypeReplace:
		return Replace(obj1, patch)
	case config.PatchTypeRemove:
		return Remove(obj1, patch)
	case config.PatchTypeAdd:
		return Add(obj1, patch)
	case config.PatchTypeCopyFromObject:
		return CopyFromObject(obj1, obj2, patch)
	}

	return fmt.Errorf("patch operation is missing or is not recognized (%s)", patch.Operation)
}

func NewNodeFromString(in string) (*yaml.Node, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(in), &node)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%w", in, err)
	}

	return &node, nil
}

func NewNode(raw interface{}) (*yaml.Node, error) {
	doc, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling struct: %+v\n\n%w", raw, err)
	}

	var node yaml.Node
	err = yaml.Unmarshal(doc, &node)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%w", string(doc), err)
	}

	return &node, nil
}

func NewJSONNode(raw interface{}) (*yaml.Node, error) {
	doc, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling struct: %+v\n\n%w", raw, err)
	}

	var node yaml.Node
	err = yaml.Unmarshal(doc, &node)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%w", string(doc), err)
	}

	return &node, nil
}
