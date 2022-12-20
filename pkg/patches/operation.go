package patches

import (
	"github.com/pkg/errors"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	yaml "gopkg.in/yaml.v3"
)

func ReplaceNode(doc *yaml.Node, match *yaml.Node, value *yaml.Node) {
	parent := Find(doc, ContainsChild(match))
	if parent != nil {
		parent.Content = replaceChildAtIndex(parent, match, value)
	}
}

func Find(doc *yaml.Node, predicate func(*yaml.Node) bool) *yaml.Node {
	if predicate(doc) {
		return doc
	}

	for _, content := range doc.Content {
		if found := Find(content, predicate); found != nil {
			return found
		}
	}

	return nil
}

func ContainsChild(child *yaml.Node) func(*yaml.Node) bool {
	return func(node *yaml.Node) bool {
		for _, c := range node.Content {
			if c == child {
				return true
			}
		}
		return false
	}
}

func ChildIndex(children []*yaml.Node, child *yaml.Node) int {
	for p, v := range children {
		if v == child {
			return p
		}
	}
	return -1
}

func removeProperty(parent *yaml.Node, child *yaml.Node) []*yaml.Node {
	childIndex := ChildIndex(parent.Content, child)
	return append(parent.Content[0:childIndex-1], parent.Content[childIndex+1:]...)
}

func removeChild(parent *yaml.Node, child *yaml.Node) []*yaml.Node {
	var remaining []*yaml.Node
	for _, current := range parent.Content {
		if child == current {
			continue
		}
		remaining = append(remaining, current)
	}
	return remaining
}

func AddChildAtIndex(parent *yaml.Node, child *yaml.Node, value *yaml.Node) []*yaml.Node {
	childIdx := ChildIndex(parent.Content, child)
	return append(parent.Content[0:childIdx], append(value.Content, parent.Content[childIdx:]...)...)
}

func replaceChildAtIndex(parent *yaml.Node, child *yaml.Node, value *yaml.Node) []*yaml.Node {
	childIdx := ChildIndex(parent.Content, child)
	return append(parent.Content[0:childIdx], append(value.Content, parent.Content[childIdx+1:]...)...)
}

func FindMatches(doc *yaml.Node, path string) ([]*yaml.Node, error) {
	yamlPath, err := yamlpath.NewPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "parsing path")
	}

	return yamlPath.Find(doc)
}

func getParents(doc *yaml.Node, path OpPath) ([]*yaml.Node, error) {
	parentPath, err := yamlpath.NewPath(path.getParentPath())
	if err != nil {
		return nil, err
	}

	parents, err := parentPath.Find(doc)
	if err != nil {
		return nil, err
	}

	return parents, nil
}
