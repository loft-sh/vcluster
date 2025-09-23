package namespaces

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets" // Ensure this import is present
)

func ValidateNamespaceSyncConfig(c *config.Config, name, namespace string) error {
	if !c.Sync.ToHost.Namespaces.Enabled {
		return nil
	}

	configPathIdentifier := "config.sync.toHost.namespaces.mappings.byName"

	if len(c.Sync.ToHost.Namespaces.Mappings.ByName) == 0 {
		return fmt.Errorf("%s are empty", configPathIdentifier)
	}

	virtualNamespaceSet := sets.NewString()
	hostNamespaceSet := sets.NewString()

	// for each vnamespace:hostNamespace mapping
	for vNS, hNS := range c.Sync.ToHost.Namespaces.Mappings.ByName {
		// first check for duplicate entries
		if virtualNamespaceSet.Has(vNS) {
			return fmt.Errorf("%s: duplicate virtual namespace '%s' found in mappings", configPathIdentifier, vNS)
		}
		virtualNamespaceSet.Insert(vNS)

		if hostNamespaceSet.Has(hNS) {
			return fmt.Errorf("%s: duplicate host namespace '%s' found in mappings", configPathIdentifier, hNS)
		}
		hostNamespaceSet.Insert(hNS)

		// then check for matching patterns
		vIsPattern := IsPattern(vNS)
		hIsPattern := IsPattern(hNS)

		if vIsPattern != hIsPattern {
			return fmt.Errorf("%s: '%s':'%s' has mismatched wildcard '*' usage - pattern must always map to another pattern", configPathIdentifier, vNS, hNS)
		}

		// check common rules for vns and hns sides
		var errLoop error
		if vIsPattern && hIsPattern {
			// validate pattern mapping rule
			errLoop = validateToHostPatternNamespaceMapping(vNS, hNS, name, configPathIdentifier)
		} else {
			// validate exact mapping rule
			errLoop = validateToHostExactNamespaceMapping(vNS, hNS, name, configPathIdentifier)
		}
		if errLoop != nil {
			return errLoop
		}

		// validate hns rules
		if err := validateHostMappingRules(hNS, hIsPattern, configPathIdentifier, name, namespace); err != nil {
			return err
		}
	}
	return nil
}

func validateToHostExactNamespaceMapping(vNS, hNS, vclusterName, configPathIdentifier string) error {
	if err := validateToHostExactNamespaceMappingPart(vNS, vclusterName, "virtual namespace", configPathIdentifier); err != nil {
		return err
	}

	if err := validateToHostExactNamespaceMappingPart(hNS, vclusterName, "host namespace", configPathIdentifier); err != nil { //nolint:revive
		return err
	}
	return nil
}

func validateToHostPatternNamespaceMapping(vNS, hNS, vclusterName, configPathIdentifier string) error {
	if err := validateToHostPatternNamespaceMappingPart(vNS, vclusterName, "virtual namespace", configPathIdentifier); err != nil {
		return err
	}

	if err := validateToHostPatternNamespaceMappingPart(hNS, vclusterName, "host namespace", configPathIdentifier); err != nil { //nolint:revive
		return err
	}
	return nil
}

func validateToHostExactNamespaceMappingPart(name, vclusterName, partIdentifier, configPathIdentifier string) error {
	if name == "" {
		return fmt.Errorf("%s: %s cannot be empty", configPathIdentifier, partIdentifier)
	}
	if IsPattern(name) {
		return fmt.Errorf("%s: %s '%s' is treated as exact but contains a wildcard '*'", configPathIdentifier, partIdentifier, name)
	}

	if err := validateNamePlaceholderUsage(name, vclusterName, partIdentifier, configPathIdentifier); err != nil {
		return err
	}

	nameForValidation := name
	if strings.Contains(name, NamePlaceholder) {
		nameForValidation = strings.ReplaceAll(name, NamePlaceholder, vclusterName)
	}

	errs := validation.ValidateNamespaceName(nameForValidation, false)
	if len(errs) > 0 {
		return fmt.Errorf("%s: invalid %s name '%s': %v", configPathIdentifier, partIdentifier, name, errs[0])
	}
	return nil
}

func validateToHostPatternNamespaceMappingPart(pattern, vclusterName, partIdentifier, configPathIdentifier string) error {
	if !IsPattern(pattern) {
		return fmt.Errorf("%s: %s '%s' is treated as a pattern but does not contain a wildcard '*'", configPathIdentifier, partIdentifier, pattern)
	}

	if strings.Count(pattern, WildcardChar) != 1 {
		return fmt.Errorf("%s: %s pattern '%s' must contain exactly one '*'", configPathIdentifier, partIdentifier, pattern)
	}

	if !strings.HasSuffix(pattern, WildcardChar) {
		return fmt.Errorf("%s: %s pattern '%s' must have the wildcard '*' at the end", configPathIdentifier, partIdentifier, pattern)
	}

	prefix := strings.TrimSuffix(pattern, WildcardChar)

	if err := validateNamePlaceholderUsage(prefix, vclusterName, fmt.Sprintf("%s pattern prefix", partIdentifier), configPathIdentifier); err != nil {
		return fmt.Errorf("%w (from pattern '%s')", err, pattern)
	}

	literalPrefixForValidation := strings.ReplaceAll(prefix, NamePlaceholder, vclusterName)

	if len(literalPrefixForValidation) == 0 {
		// This is a case where we're handling a catch-all '*' pattern - since we removed the wildcard suffix now we're working with empty string
		return nil
	}

	if len(literalPrefixForValidation) > 32 {
		return fmt.Errorf("%s: literal parts of %s pattern prefix '%s' (from '%s') cannot be longer than 32 characters (literal length: %d)", configPathIdentifier, partIdentifier, prefix, pattern, len(literalPrefixForValidation))
	}

	errs := validation.ValidateNamespaceName(literalPrefixForValidation, true)
	if len(errs) > 0 {
		return fmt.Errorf("%s: invalid %s pattern '%s': %s", configPathIdentifier, partIdentifier, pattern, errs[0])
	}

	return nil
}

func validateNamePlaceholderUsage(namePart, vclusterName, partTypeIdentifier, configPathIdentifier string) error {
	if !strings.Contains(namePart, NamePlaceholder) {
		if strings.Contains(namePart, "${") && strings.Contains(namePart, "}") {
			return fmt.Errorf("%s: %s '%s' contains an unsupported placeholder; only '%s' is allowed", configPathIdentifier, partTypeIdentifier, namePart, NamePlaceholder)
		}
		return nil
	}

	if strings.Count(namePart, NamePlaceholder) > 1 {
		return fmt.Errorf("%s: %s '%s' contains placeholder '%s' multiple times", configPathIdentifier, partTypeIdentifier, namePart, NamePlaceholder)
	}

	tempName := strings.ReplaceAll(namePart, NamePlaceholder, vclusterName)
	if strings.Contains(tempName, "${") && strings.Contains(tempName, "}") {
		return fmt.Errorf("%s: %s '%s' contains an unsupported placeholder; only a single '%s' is allowed", configPathIdentifier, partTypeIdentifier, namePart, NamePlaceholder)
	}

	return nil
}

func validateHostMappingRules(hNS string, hIsPattern bool, configPathIdentifier, vclusterName, vclusterNamespace string) error {
	// explicitly check against wildcard "catch all" mapping
	if hIsPattern && hNS == WildcardChar {
		return fmt.Errorf("%s: host pattern mappings must use a prefix before wildcard: %s", configPathIdentifier, hNS)
	}

	// validate we're not mapping to host namespace in which vcluster is running
	if err := validateHostMappingNotControlPlane(hNS, hIsPattern, configPathIdentifier, vclusterName, vclusterNamespace); err != nil {
		return err
	}

	return nil
}

func validateHostMappingNotControlPlane(hNS string, hIsPattern bool, configPathIdentifier, vclusterName, vclusterNamespace string) error {
	if hIsPattern {
		resolvedPatternString := ProcessNamespaceName(hNS, vclusterName)
		_, matched := MatchAndExtractWildcard(vclusterNamespace, resolvedPatternString)
		if matched {
			return fmt.Errorf("%s: host namespace pattern '%s' conflicts with control plane namespace '%s'", configPathIdentifier, hNS, vclusterNamespace)
		}
	} else {
		resolvedHNS := ProcessNamespaceName(hNS, vclusterName)
		if resolvedHNS == vclusterNamespace {
			return fmt.Errorf("%s: host namespace mapping '%s' conflicts with control plane namespace '%s'", configPathIdentifier, hNS, vclusterNamespace)
		}
	}
	return nil
}
