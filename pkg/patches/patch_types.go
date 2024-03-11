package patches

import (
	"fmt"
	"strconv"

	"github.com/loft-sh/vcluster/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

func CopyFromObject(obj1, obj2 *yaml.Node, patch *config.Patch) error {
	if obj2 == nil {
		return nil
	}

	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	fromPath := patch.FromPath
	if fromPath == "" {
		fromPath = patch.Path
	}
	fromMatches, err := FindMatches(obj2, fromPath)
	if err != nil {
		return errors.Wrap(err, "find from matches")
	} else if len(fromMatches) > 1 {
		return fmt.Errorf("more than 1 match found for path %s", fromPath)
	}

	if len(fromMatches) == 1 && len(matches) == 0 {
		validated, err := ValidateAllConditions(obj1, nil, patch.Conditions)
		if err != nil {
			return errors.Wrap(err, "validate conditions")
		} else if !validated {
			return nil
		}

		return createPath(obj1, patch.Path, fromMatches[0])
	}

	for _, m := range matches {
		validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
		if err != nil {
			return errors.Wrap(err, "validate conditions")
		} else if !validated {
			continue
		}

		if len(fromMatches) == 1 {
			ReplaceNode(obj1, m, &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					fromMatches[0],
				},
			})
		} else {
			parent := Find(obj1, ContainsChild(m))
			removeChild(parent, m)
		}
	}

	return nil
}

func Remove(obj1 *yaml.Node, patch *config.Patch) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	for _, m := range matches {
		validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
		if err != nil {
			return errors.Wrap(err, "validate conditions")
		} else if !validated {
			continue
		}

		parent := Find(obj1, ContainsChild(m))
		switch parent.Kind {
		case yaml.MappingNode:
			parent.Content = removeProperty(parent, m)
		case yaml.SequenceNode:
			parent.Content = removeChild(parent, m)
		case yaml.DocumentNode, yaml.ScalarNode, yaml.AliasNode:
		}
	}

	return nil
}

func Add(obj1 *yaml.Node, patch *config.Patch) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	value, err := NewNode(patch.Value)
	if err != nil {
		return errors.Wrap(err, "new node from value")
	}

	if len(matches) == 0 {
		validated, err := ValidateAllConditions(obj1, nil, patch.Conditions)
		if err != nil {
			return errors.Wrap(err, "validate conditions")
		} else if !validated {
			return nil
		}

		err = createPath(obj1, patch.Path, value)
		if err != nil {
			return err
		}
	} else {
		for _, m := range matches {
			validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
			if err != nil {
				return errors.Wrap(err, "validate conditions")
			} else if !validated {
				continue
			}

			AddNode(obj1, m, value)
		}
	}

	return nil
}

func Replace(obj1 *yaml.Node, patch *config.Patch) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	value, err := NewNode(patch.Value)
	if err != nil {
		return errors.Wrap(err, "new node from value")
	}

	for _, m := range matches {
		validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
		if err != nil {
			return errors.Wrap(err, "validate conditions")
		} else if !validated {
			continue
		}

		ReplaceNode(obj1, m, value)
	}

	return nil
}

func RewriteName(obj1 *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	for _, m := range matches {
		switch m.Kind {
		case yaml.ScalarNode:
			err = ValidateAndTranslateName(obj1, m, patch, resolver, "")
		case yaml.SequenceNode:
			for _, subNode := range m.Content {
				err = ProcessRewrite(subNode, patch, resolver)

				if err != nil {
					return err
				}
			}
		case yaml.MappingNode:
			err = ProcessRewrite(m, patch, resolver)
		case yaml.DocumentNode, yaml.AliasNode:
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func ProcessRewrite(obj *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	var namespace string
	var err error

	if patch.NamespacePath != "" {
		namespace, err = GetNamespace(obj, patch)

		if err != nil {
			return err
		}
	}

	nameMatches, err := FindMatches(obj, patch.NamePath)
	if err != nil {
		return errors.Wrap(err, "find name matches")
	}

	for _, nameMatch := range nameMatches {
		if nameMatch.Kind != yaml.ScalarNode {
			continue
		}
		err = ValidateAndTranslateName(obj, nameMatch, patch, resolver, namespace)

		if err != nil {
			return err
		}
	}

	// Translate namespace
	if namespace != "" {
		namespaceMatches, err := FindMatches(obj, patch.NamespacePath)
		if err != nil {
			return errors.Wrap(err, "find namespace matches")
		}

		for _, namespaceMatch := range namespaceMatches {
			if namespaceMatch.Kind != yaml.ScalarNode {
				continue
			}
			err = ValidateAndTranslateNamespace(obj, namespaceMatch, patch, resolver)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateAndTranslateName(obj *yaml.Node, match *yaml.Node, patch *config.Patch, resolver NameResolver, namespace string) error {
	validated, err := ValidateAllConditions(obj, match, patch.Conditions)
	if err != nil {
		return errors.Wrap(err, "validate conditions")
	} else if !validated {
		return nil
	}

	var translatedName string

	if namespace != "" {
		translatedName, err = resolver.TranslateNameWithNamespace(match.Value, namespace, patch.ParsedRegex, patch.FromPath)
	} else {
		translatedName, err = resolver.TranslateName(match.Value, patch.ParsedRegex, patch.FromPath)
	}

	if err != nil {
		return err
	}

	newNode, err := NewNode(translatedName)
	if err != nil {
		return errors.Wrap(err, "create node")
	}

	ReplaceNode(obj, match, newNode)

	return nil
}

func ValidateAndTranslateNamespace(obj *yaml.Node, match *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	validated, err := ValidateAllConditions(obj, match, patch.Conditions)
	if err != nil {
		return errors.Wrap(err, "validate conditions")
	} else if !validated {
		return nil
	}

	translatedNamespace, err := resolver.TranslateNamespaceRef(match.Value)
	if err != nil {
		return err
	}

	newNode, err := NewNode(translatedNamespace)
	if err != nil {
		return errors.Wrap(err, "create node")
	}

	ReplaceNode(obj, match, newNode)

	return nil
}

func GetNamespace(obj *yaml.Node, patch *config.Patch) (string, error) {
	var namespace string

	matches, err := FindMatches(obj, patch.NamespacePath)
	if err != nil {
		return namespace, errors.Wrap(err, "find matches namespace")
	}

	if len(matches) > 1 {
		return namespace, errors.New("found multiple namespace references")
	}

	for _, m := range matches {
		namespace = m.Value
	}

	return namespace, nil
}

func RewriteLabelKey(obj1 *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	for _, m := range matches {
		if m.Kind == yaml.ScalarNode {
			validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
			if err != nil {
				return errors.Wrap(err, "validate conditions")
			} else if !validated {
				continue
			}

			labelKey := m.Value
			if labelKey == "" {
				continue
			}

			// translate label selector
			labelKey, err = resolver.TranslateLabelKey(labelKey)
			if err != nil {
				return err
			}

			newNode, err := NewNode(labelKey)
			if err != nil {
				return errors.Wrap(err, "create node")
			}

			ReplaceNode(obj1, m, newNode)
		}
	}

	return nil
}

func RewriteLabelSelector(obj1 *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	for _, m := range matches {
		if m.Kind == yaml.MappingNode {
			validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
			if err != nil {
				return errors.Wrap(err, "validate conditions")
			} else if !validated {
				continue
			}

			yamlString, err := yaml.Marshal(m)
			if err != nil {
				return errors.Wrap(err, "marshal label selector")
			}

			// try to unmarshal into label selector first
			var newNode *yaml.Node
			labelSelector := map[string]string{}
			err = k8syaml.UnmarshalStrict(yamlString, &labelSelector)
			if err != nil {
				return errors.Wrap(err, "unmarshal label selector")
			}

			// translate label selector
			labelSelector, err = resolver.TranslateLabelSelector(labelSelector)
			if err != nil {
				return err
			}

			newNode, err = NewNode(labelSelector)
			if err != nil {
				return errors.Wrap(err, "create node")
			}

			ReplaceNode(obj1, m, newNode)
		}
	}

	return nil
}

func RewriteLabelExpressionsSelector(obj1 *yaml.Node, patch *config.Patch, resolver NameResolver) error {
	matches, err := FindMatches(obj1, patch.Path)
	if err != nil {
		return errors.Wrap(err, "find matches")
	}

	for _, m := range matches {
		if m.Kind == yaml.MappingNode {
			validated, err := ValidateAllConditions(obj1, m, patch.Conditions)
			if err != nil {
				return errors.Wrap(err, "validate conditions")
			} else if !validated {
				continue
			}

			yamlString, err := yaml.Marshal(m)
			if err != nil {
				return errors.Wrap(err, "marshal label selector")
			}

			// try to unmarshal into label selector first
			var newNode *yaml.Node
			labelSelector := &metav1.LabelSelector{}
			// using sigs.k8s.io/yaml below because it can unmarshal based on json tags,
			// because yaml tags are not present on the metav1.LabelSelector fields
			err = k8syaml.Unmarshal(yamlString, labelSelector)
			if err != nil {
				return errors.Wrap(err, "unmarshal label selector")
			}

			// translate label expressions selector
			labelSelector, err = resolver.TranslateLabelExpressionsSelector(labelSelector)
			if err != nil {
				return err
			}

			// using NewJSONNode below because yaml tags are not present on the metav1.LabelSelector fields
			newNode, err = NewJSONNode(labelSelector)
			if err != nil {
				return errors.Wrap(err, "create node")
			}

			ReplaceNode(obj1, m, newNode)
		}
	}

	return nil
}

func createPath(obj1 *yaml.Node, path string, value *yaml.Node) error {
	// unpack document nodes
	if value != nil && value.Kind == yaml.DocumentNode {
		value = value.Content[0]
	}

	opPath := OpPath(path)
	matches, err := getParents(obj1, opPath)
	if err != nil {
		return fmt.Errorf("could not replace using path: %s", path)
	} else if len(matches) == 0 {
		// are we at the top path?
		parentPath := opPath.getParentPath()
		if path == parentPath || parentPath == "" || path == "" {
			return nil
		}

		if isSequenceChild(path) {
			value = createSequenceNode(path, value)
		} else {
			value = createMappingNode(path, value)
		}

		return createPath(obj1, parentPath, value)
	}

	// check if we expect an array or map as parent
	for _, match := range matches {
		parent := Find(obj1, ContainsChild(match))
		switch match.Kind {
		case yaml.ScalarNode:
			parent.Content = AddChildAtIndex(parent, match, value)
		case yaml.MappingNode:
			match.Content = append(match.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: opPath.getChildName(),
			}, value)
		case yaml.SequenceNode:
			match.Content = append(match.Content, value)
		case yaml.DocumentNode:
			match.Content[0].Content = append(match.Content[0].Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: opPath.getChildName(),
			}, value)
		case yaml.AliasNode:
			// TODO: Implement node aliases in the future
		}
	}

	return nil
}

func isSequenceChild(path string) bool {
	opPath := OpPath(path)
	propertyName := opPath.getChildName()
	if propertyName == "" {
		return false
	}
	_, err := strconv.Atoi(propertyName)
	return err == nil
}

func createSequenceNode(_ string, child *yaml.Node) *yaml.Node {
	childNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
	}
	if child != nil {
		childNode.Content = append(
			childNode.Content,
			child,
		)
	}
	return childNode
}

func createMappingNode(path string, child *yaml.Node) *yaml.Node {
	opPath := OpPath(path)
	childNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	if child != nil {
		childNode.Content = append(
			childNode.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: opPath.getChildName(),
				Tag:   "!!str",
			},
			child,
		)
	}

	return childNode
}

func AddNode(obj1 *yaml.Node, match *yaml.Node, value *yaml.Node) {
	parent := Find(obj1, ContainsChild(match))
	switch match.Kind {
	case yaml.ScalarNode:
		parent.Content = AddChildAtIndex(parent, match, value)
	case yaml.MappingNode:
		match.Content = append(match.Content, value.Content[0].Content...)
	case yaml.SequenceNode:
		match.Content = append(match.Content, value.Content...)
	case yaml.DocumentNode:
		match.Content[0].Content = append(match.Content[0].Content, value.Content[0].Content...)
	case yaml.AliasNode:
		// TODO: Implement node aliases in the future
	}
}
