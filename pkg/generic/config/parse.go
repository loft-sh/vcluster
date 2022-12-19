package config

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func Parse(rawConfig string) (*Config, error) {
	c := &Config{}
	err := yaml.UnmarshalStrict([]byte(rawConfig), c)
	if err != nil {
		return nil, err
	}

	err = validate(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func validate(config *Config) error {
	if config.Version != Version {
		return fmt.Errorf("unsupported configuration version. Only %s is supported by this plugin version", config.Version)
	}

	for idx, exp := range config.Exports {
		if exp == nil {
			return fmt.Errorf("exports[%d] is required", idx)
		}

		if exp.Kind == "" {
			return fmt.Errorf("exports[%d].kind is required", idx)
		}

		if exp.APIVersion == "" {
			return fmt.Errorf("exports[%d].APIVersion is required", idx)
		}

		for patchIdx, patch := range exp.Patches {
			err := validatePatch(patch)
			if err != nil {
				return errors.Wrapf(err, "exports[%d].patches[%d]", idx, patchIdx)
			}
		}

		for patchIdx, patch := range exp.ReversePatches {
			err := validatePatch(patch)
			if err != nil {
				return errors.Wrapf(err, "exports[%d].reversPatches[%d]", idx, patchIdx)
			}
		}
	}

	return nil
}

func validatePatch(patch *Patch) error {
	switch patch.Operation {
	case PatchTypeRemove, PatchTypeReplace, PatchTypeAdd:
		if patch.FromPath != "" {
			return fmt.Errorf("fromPath is not supported for this operation")
		}

		return nil
	case PatchTypeRewriteName, PatchTypeRewriteLabelKey, PatchTypeRewriteLabelSelector, PatchTypeRewriteLabelExpressionsSelector:
		return nil
	case PatchTypeCopyFromObject:
		if patch.FromPath == "" {
			return fmt.Errorf("fromPath is required for this operation")
		}

		return nil
	default:
		return fmt.Errorf("unsupported patch type %s", patch.Operation)
	}
}
