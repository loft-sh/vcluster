package patches

import (
	"github.com/loft-sh/vcluster/config"
	"github.com/pkg/errors"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	yaml "gopkg.in/yaml.v3"
)

func ValidateAllConditions(obj *yaml.Node, match *yaml.Node, conditions []*config.PatchCondition) (bool, error) {
	for _, condition := range conditions {
		matched, err := ValidateCondition(obj, match, condition)
		if err != nil {
			return false, err
		} else if !matched {
			return false, nil
		}
	}

	return true, nil
}

func ValidateCondition(obj *yaml.Node, match *yaml.Node, condition *config.PatchCondition) (bool, error) {
	if condition == nil {
		return true, nil
	}

	var matches []*yaml.Node
	if condition.SubPath != "" {
		if match == nil {
			if (condition.Empty != nil && *condition.Empty) || condition.NotEqual != nil {
				return true, nil
			}

			return false, nil
		}

		path, err := yamlpath.NewPath(condition.SubPath)
		if err != nil {
			return false, errors.Wrap(err, "parsing sub path")
		}

		matches, err = path.Find(match)
		if err != nil {
			return false, errors.Wrap(err, "find matches")
		}
	} else if condition.Path != "" {
		path, err := yamlpath.NewPath(condition.Path)
		if err != nil {
			return false, errors.Wrap(err, "parsing path")
		}

		matches, err = path.Find(obj)
		if err != nil {
			return false, errors.Wrap(err, "find matches")
		}
	}

	// no matches
	if len(matches) == 0 {
		if (condition.Empty != nil && *condition.Empty) || condition.NotEqual != nil {
			return true, nil
		}

		return false, nil
	}

	// only one match needs to fulfill our condition
	for _, match := range matches {
		stringValue := match.Value
		if match.Kind != yaml.ScalarNode {
			stringValue = getStringValue(match)
		}

		if condition.Empty != nil {
			if *condition.Empty && stringValue == "" {
				return true, nil
			} else if !*condition.Empty && stringValue != "" {
				return true, nil
			}

			continue
		} else if condition.Equal != nil {
			if getStringValue(condition.Equal) == stringValue {
				return true, nil
			}

			continue
		} else if condition.NotEqual != nil {
			if getStringValue(condition.NotEqual) != stringValue {
				return true, nil
			}

			continue
		}
	}

	return false, nil
}

func getStringValue(value interface{}) string {
	strValue, ok := value.(string)
	if ok {
		return strValue
	}

	out, _ := yaml.Marshal(value)
	return string(out)
}
