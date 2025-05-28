package namespaces

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"k8s.io/apimachinery/pkg/api/validation"
)

func ValidateNamespaceSyncConfig(c *config.Config, name, namespace string) error {
	if !c.Sync.ToHost.Namespaces.Enabled {
		return nil
	}

	configPathIdentifier := "config.sync.toHost.namespaces.mappings.byName"

	if len(c.Sync.ToHost.Namespaces.Mappings.ByName) == 0 {
		return fmt.Errorf("%s are empty", configPathIdentifier)
	}

	virtualNamespaceNames := make([]string, 0, len(c.Sync.ToHost.Namespaces.Mappings.ByName))
	hostNamespaceNames := make([]string, 0, len(c.Sync.ToHost.Namespaces.Mappings.ByName))

	for vNS, hNS := range c.Sync.ToHost.Namespaces.Mappings.ByName {
		virtualNamespaceNames = append(virtualNamespaceNames, vNS)
		hostNamespaceNames = append(hostNamespaceNames, hNS)
	}

	if err := validateNoDuplicatedMappingKeys(virtualNamespaceNames, "virtual namespace", configPathIdentifier); err != nil {
		return err
	}

	if err := validateNoDuplicatedMappingKeys(hostNamespaceNames, "host namespace", configPathIdentifier); err != nil {
		return err
	}

	for vNS, hNS := range c.Sync.ToHost.Namespaces.Mappings.ByName {
		vIsPattern := IsPattern(vNS)
		hIsPattern := IsPattern(hNS)

		if vIsPattern != hIsPattern {
			return fmt.Errorf("%s: '%s':'%s' has mismatched wildcard '*' usage - pattern must always map to another pattern", configPathIdentifier, vNS, hNS)
		}

		if err := validateHostMappingNotControlPlane(hNS, hIsPattern, configPathIdentifier, name, namespace); err != nil {
			return err
		}

		var errLoop error // Renamed to avoid shadowing outer err
		if vIsPattern && hIsPattern {
			errLoop = validateToHostPatternNamespaceMapping(vNS, hNS, name, configPathIdentifier)
		} else {
			errLoop = validateToHostExactNamespaceMapping(vNS, hNS, name, configPathIdentifier)
		}
		if errLoop != nil {
			return errLoop
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

func validateNoDuplicatedMappingKeys(items []string, itemType string, configPathIdentifier string) error {
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			return fmt.Errorf("%s: duplicate %s '%s' found in mappings", configPathIdentifier, itemType, item)
		}
		seen[item] = true
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
