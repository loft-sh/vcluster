package config

import (
	"fmt"

	"github.com/samber/lo"
	"sigs.k8s.io/yaml"
)

var (
	verbs = []string{"get", "list", "create", "update", "patch", "watch", "delete", "deletecollection"}
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
		return fmt.Errorf("unsupported configuration version. Only %s is supported currently", Version)
	}

	err := validateExportDuplicates(config.Exports)
	if err != nil {
		return err
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
				return fmt.Errorf("invalid exports[%d].patches[%d]: %w", idx, patchIdx, err)
			}
		}

		for patchIdx, patch := range exp.ReversePatches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid exports[%d].reversPatches[%d]: %w", idx, patchIdx, err)
			}
		}
	}

	err = validateImportDuplicates(config.Imports)
	if err != nil {
		return err
	}

	for idx, imp := range config.Imports {
		if imp == nil {
			return fmt.Errorf("imports[%d] is required", idx)
		}

		if imp.Kind == "" {
			return fmt.Errorf("imports[%d].kind is required", idx)
		}

		if imp.APIVersion == "" {
			return fmt.Errorf("imports[%d].APIVersion is required", idx)
		}

		for patchIdx, patch := range imp.Patches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid imports[%d].patches[%d]: %w", idx, patchIdx, err)
			}
		}

		for patchIdx, patch := range imp.ReversePatches {
			err := validatePatch(patch)
			if err != nil {
				return fmt.Errorf("invalid imports[%d].reversPatches[%d]: %w", idx, patchIdx, err)
			}
		}
	}

	if config.Hooks != nil {
		// HostToVirtual validation
		for idx, hook := range config.Hooks.HostToVirtual {
			for idy, verb := range hook.Verbs {
				if err := validateVerb(verb); err != nil {
					return fmt.Errorf("invalid hooks.hostToVirtual[%d].verbs[%d]: %w", idx, idy, err)
				}
			}

			for idy, patch := range hook.Patches {
				if err := validatePatch(patch); err != nil {
					return fmt.Errorf("invalid hooks.hostToVirtual[%d].patches[%d]: %w", idx, idy, err)
				}
			}
		}

		// VirtualToHost validation
		for idx, hook := range config.Hooks.VirtualToHost {
			for idy, verb := range hook.Verbs {
				if err := validateVerb(verb); err != nil {
					return fmt.Errorf("invalid hooks.virtualToHost[%d].verbs[%d]: %w", idx, idy, err)
				}
			}

			for idy, patch := range hook.Patches {
				if err := validatePatch(patch); err != nil {
					return fmt.Errorf("invalid hooks.virtualToHost[%d].patches[%d]: %w", idx, idy, err)
				}
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
	case PatchTypeRewriteName, PatchTypeRewriteLabelSelector, PatchTypeRewriteLabelKey, PatchTypeRewriteLabelExpressionsSelector:
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

func validateVerb(verb string) error {
	if !lo.Contains(verbs, verb) {
		return fmt.Errorf("invalid verb \"%s\"; expected on of %q", verb, verbs)
	}

	return nil
}

func validateExportDuplicates(exports []*Export) error {
	gvks := map[string]bool{}
	for _, e := range exports {
		k := fmt.Sprintf("%s|%s", e.APIVersion, e.Kind)
		_, found := gvks[k]
		if found {
			return fmt.Errorf("duplicate export for APIVersion %s and %s Kind, only one export for each APIVersion+Kind is permitted", e.APIVersion, e.Kind)
		}
		gvks[k] = true
	}

	return nil
}

func validateImportDuplicates(imports []*Import) error {
	gvks := map[string]bool{}
	for _, e := range imports {
		k := fmt.Sprintf("%s|%s", e.APIVersion, e.Kind)
		_, found := gvks[k]
		if found {
			return fmt.Errorf("duplicate import for APIVersion %s and %s Kind, only one import for each APIVersion+Kind is permitted", e.APIVersion, e.Kind)
		}
		gvks[k] = true
	}

	return nil
}
