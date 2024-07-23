package config

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/validation"
)

var allowedPodSecurityStandards = map[string]bool{
	"privileged": true,
	"baseline":   true,
	"restricted": true,
}

var (
	verbs = []string{"get", "list", "create", "update", "patch", "watch", "delete", "deletecollection"}
)

func ValidateConfigAndSetDefaults(config *VirtualClusterConfig) error {
	// check the value of pod security standard
	if config.Policies.PodSecurityStandard != "" && !allowedPodSecurityStandards[config.Policies.PodSecurityStandard] {
		return fmt.Errorf("invalid argument enforce-pod-security-standard=%s, must be one of: privileged, baseline, restricted", config.Policies.PodSecurityStandard)
	}

	// parse tolerations
	for _, t := range config.Sync.ToHost.Pods.EnforceTolerations {
		_, err := toleration.ParseToleration(t)
		if err != nil {
			return err
		}
	}

	// check if enable scheduler works correctly
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && !config.Sync.FromHost.Nodes.Selector.All && len(config.Sync.FromHost.Nodes.Selector.Labels) == 0 {
		config.Sync.FromHost.Nodes.Selector.All = true
	}

	// enable additional controllers required for scheduling with storage
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && config.Sync.ToHost.PersistentVolumeClaims.Enabled {
		if config.Sync.FromHost.CSINodes.Enabled == "auto" {
			config.Sync.FromHost.CSINodes.Enabled = "true"
		}
		if config.Sync.FromHost.CSIStorageCapacities.Enabled == "auto" {
			config.Sync.FromHost.CSIStorageCapacities.Enabled = "true"
		}
		if config.Sync.FromHost.CSIDrivers.Enabled == "auto" {
			config.Sync.FromHost.CSIDrivers.Enabled = "true"
		}
		if config.Sync.FromHost.StorageClasses.Enabled == "auto" && !config.Sync.ToHost.StorageClasses.Enabled {
			config.Sync.FromHost.StorageClasses.Enabled = "true"
		}
	}

	// check if nodes controller needs to be enabled
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && !config.Sync.FromHost.Nodes.Enabled {
		return fmt.Errorf("sync.fromHost.nodes.enabled is false, but required if using virtual scheduler")
	}

	// check if storage classes and host storage classes are enabled at the same time
	if config.Sync.FromHost.StorageClasses.Enabled == "true" && config.Sync.ToHost.StorageClasses.Enabled {
		return fmt.Errorf("you cannot enable both sync.fromHost.storageClasses.enabled and sync.toHost.storageClasses.enabled at the same time. Choose only one of them")
	}

	// validate central admission control
	err := validateCentralAdmissionControl(config)
	if err != nil {
		return err
	}

	// validate generic sync config
	err = validateGenericSyncConfig(config.Experimental.GenericSync)
	if err != nil {
		return fmt.Errorf("validate experimental.genericSync")
	}

	// validate distro
	err = validateDistro(config)
	if err != nil {
		return err
	}

	err = validateK0sAndNoExperimentalKubeconfig(config)
	if err != nil {
		return err
	}

	// check deny proxy requests
	for _, c := range config.Experimental.DenyProxyRequests {
		err := validateCheck(c)
		if err != nil {
			return err
		}
	}

	// check resolve dns
	err = validateMappings(config.Networking.ResolveDNS)
	if err != nil {
		return err
	}

	// set service name
	if config.ControlPlane.Advanced.WorkloadServiceAccount.Name == "" {
		config.ControlPlane.Advanced.WorkloadServiceAccount.Name = "vc-workload-" + config.Name
	}

	return nil
}

func validateDistro(config *VirtualClusterConfig) error {
	enabledDistros := 0
	if config.Config.ControlPlane.Distro.K3S.Enabled {
		enabledDistros++
	}
	if config.Config.ControlPlane.Distro.K0S.Enabled {
		enabledDistros++
	}
	if config.Config.ControlPlane.Distro.K8S.Enabled {
		enabledDistros++
	}

	if enabledDistros > 1 {
		return fmt.Errorf("only one distribution can be enabled")
	}
	return nil
}

func validateGenericSyncConfig(config config.ExperimentalGenericSync) error {
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

func validatePatch(patch *config.Patch) error {
	switch patch.Operation {
	case config.PatchTypeRemove, config.PatchTypeReplace, config.PatchTypeAdd:
		if patch.FromPath != "" {
			return fmt.Errorf("fromPath is not supported for this operation")
		}

		return nil
	case config.PatchTypeRewriteName, config.PatchTypeRewriteLabelSelector, config.PatchTypeRewriteLabelKey, config.PatchTypeRewriteLabelExpressionsSelector:
		return nil
	case config.PatchTypeCopyFromObject:
		if patch.FromPath == "" {
			return fmt.Errorf("fromPath is required for this operation")
		}

		return nil
	default:
		return fmt.Errorf("unsupported patch type %s", patch.Operation)
	}
}

func validateVerb(verb string) error {
	if !slices.Contains(verbs, verb) {
		return fmt.Errorf("invalid verb \"%s\"; expected on of %q", verb, verbs)
	}

	return nil
}

func validateExportDuplicates(exports []*config.Export) error {
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

func validateImportDuplicates(imports []*config.Import) error {
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

func validateCentralAdmissionControl(config *VirtualClusterConfig) error {
	_, _, err := ParseExtraHooks(config.Policies.CentralAdmission.ValidatingWebhooks, config.Policies.CentralAdmission.MutatingWebhooks)
	return err
}

func ParseExtraHooks(valHooks []config.ValidatingWebhookConfiguration, mutHooks []config.MutatingWebhookConfiguration) ([]admissionregistrationv1.ValidatingWebhookConfiguration, []admissionregistrationv1.MutatingWebhookConfiguration, error) {
	decodedVal := make([]string, 0, len(valHooks))
	for _, v := range valHooks {
		bytes, err := yaml.Marshal(v)
		if err != nil {
			return nil, nil, err
		}
		decodedVal = append(decodedVal, string(bytes))
	}
	decodedMut := make([]string, 0, len(mutHooks))
	for _, v := range mutHooks {
		bytes, err := yaml.Marshal(v)
		if err != nil {
			return nil, nil, err
		}
		decodedMut = append(decodedMut, string(bytes))
	}

	validateConfs := make([]admissionregistrationv1.ValidatingWebhookConfiguration, 0, len(valHooks))
	mutateConfs := make([]admissionregistrationv1.MutatingWebhookConfiguration, 0, len(mutHooks))
	for _, v := range decodedVal {
		var valHook admissionregistrationv1.ValidatingWebhookConfiguration
		err := yaml.Unmarshal([]byte(v), &valHook)
		if err != nil {
			return nil, nil, err
		}
		for _, v := range valHook.Webhooks {
			err := validateWebhookClientCfg(v.ClientConfig)
			if err != nil {
				return nil, nil, fmt.Errorf("webhook client config was not valid for ValidatingWebhookConfiguration %s: %w", v.Name, err)
			}
		}
		validateConfs = append(validateConfs, valHook)
	}
	for _, v := range decodedMut {
		var mutHook admissionregistrationv1.MutatingWebhookConfiguration
		err := yaml.Unmarshal([]byte(v), &mutHook)
		if err != nil {
			return nil, nil, err
		}
		for _, v := range mutHook.Webhooks {
			err := validateWebhookClientCfg(v.ClientConfig)
			if err != nil {
				return nil, nil, fmt.Errorf("webhook client config was not valid for MutatingWebhookConfiguration %s: %w", v.Name, err)
			}
		}
		mutateConfs = append(mutateConfs, mutHook)
	}

	return validateConfs, mutateConfs, nil
}

func validateWebhookClientCfg(clientCfg admissionregistrationv1.WebhookClientConfig) error {
	if len(clientCfg.CABundle) != 0 {
		ok := x509.NewCertPool().AppendCertsFromPEM(clientCfg.CABundle)
		if !ok {
			return errors.New("could not parse the CABundle")
		}
	}

	if clientCfg.Service == nil && clientCfg.URL == nil {
		return errors.New("there is no service config")
	}

	if clientCfg.Service != nil && (clientCfg.Service.Name == "" || clientCfg.Service.Namespace == "") {
		return errors.New("namespace or name of the service is missing")
	}

	if clientCfg.URL != nil {
		_, err := url.Parse(*clientCfg.URL)
		if err != nil {
			return errors.New("the url was not valid")
		}
	}

	return nil
}

func validateCheck(check config.DenyRule) error {
	for _, ns := range check.Namespaces {
		errors := validation.ValidateNamespaceName(ns, false)
		if len(errors) != 0 {
			return fmt.Errorf("invalid Namespaces in %q check: %v", check.Name, errors)
		}
	}
	var err error
	for _, r := range check.Rules {
		err = validateWildcardOrExact(r.Verbs, "create", "get", "update", "patch", "delete")
		if err != nil {
			return fmt.Errorf("invalid Verb defined in the %q check: %w", check.Name, err)
		}

		err = validateWildcardOrAny(r.APIGroups)
		if err != nil {
			return fmt.Errorf("invalid APIGroup defined in the %q check: %w", check.Name, err)
		}

		err = validateWildcardOrAny(r.APIVersions)
		if err != nil {
			return fmt.Errorf("invalid APIVersion defined in the %q check: %w", check.Name, err)
		}

		if r.Scope != nil {
			switch *r.Scope {
			case string(admissionregistrationv1.ClusterScope):
			case string(admissionregistrationv1.NamespacedScope):
			case string(admissionregistrationv1.AllScopes):
			default:
				return fmt.Errorf("invalid Scope defined in the %q check: %q", check.Name, *r.Scope)
			}
		}
	}
	return nil
}

func validateWildcardOrExact(values []string, validValues ...string) error {
	if len(values) == 1 && values[0] == "*" {
		return nil
	}
	for _, val := range values {
		if val == "*" {
			return fmt.Errorf("when wildcard(*) is used, it must be the only value in the list")
		}

		// empty list of validValues means any value is valid
		valid := len(validValues) == 0
		for _, v := range validValues {
			if val == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value %q", val)
		}
	}
	return nil
}

func validateWildcardOrAny(values []string) error {
	if len(values) == 1 && values[0] == "*" {
		return nil
	}
	for _, val := range values {
		if val == "*" {
			return fmt.Errorf("when wildcard(*) is used, it must be the only value in the list")
		}
	}
	return nil
}

func validateK0sAndNoExperimentalKubeconfig(c *VirtualClusterConfig) error {
	if c.Distro() != config.K0SDistro {
		return nil
	}
	virtualclusterconfig := c.Experimental.VirtualClusterKubeConfig
	empty := config.VirtualClusterKubeConfig{}
	if virtualclusterconfig != empty {
		return errors.New("config.experimental.VirtualClusterConfig cannot be set for k0s")
	}
	return nil
}
