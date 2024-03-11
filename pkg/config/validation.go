package config

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	"github.com/samber/lo"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

var allowedPodSecurityStandards = map[string]bool{
	"privileged": true,
	"baseline":   true,
	"restricted": true,
}

var (
	verbs = []string{"get", "list", "create", "update", "patch", "watch", "delete", "deletecollection"}
)

func ValidateConfig(config *VirtualClusterConfig) error {
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
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && !config.Sync.FromHost.Nodes.SyncAll && len(config.Sync.FromHost.Nodes.Selector.Labels) == 0 {
		config.Sync.FromHost.Nodes.SyncAll = true
	}

	// enable additional controllers required for scheduling with storage
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && config.Sync.ToHost.PersistentVolumeClaims.Enabled {
		config.Sync.FromHost.CSINodes.Enabled = true
		config.Sync.FromHost.CSIStorageCapacities.Enabled = true
		config.Sync.FromHost.CSIDrivers.Enabled = true
		if !config.Sync.FromHost.StorageClasses.Enabled && !config.Sync.ToHost.StorageClasses.Enabled {
			config.Sync.FromHost.StorageClasses.Enabled = true
		}
	}

	// check if nodes controller needs to be enabled
	if config.ControlPlane.Advanced.VirtualScheduler.Enabled && !config.Sync.FromHost.Nodes.Enabled {
		return fmt.Errorf("sync.fromHost.nodes.enabled is false, but required if using virtual scheduler")
	}

	// check if storage classes and host storage classes are enabled at the same time
	if config.Sync.FromHost.StorageClasses.Enabled && config.Sync.ToHost.StorageClasses.Enabled {
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
	if config.Config.ControlPlane.Distro.EKS.Enabled {
		enabledDistros++
	}

	if enabledDistros > 1 {
		return fmt.Errorf("please only enable a single distribution")
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
	if !lo.Contains(verbs, verb) {
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

func ParseExtraHooks(valHooks, mutHooks []interface{}) ([]admissionregistrationv1.ValidatingWebhookConfiguration, []admissionregistrationv1.MutatingWebhookConfiguration, error) {
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
