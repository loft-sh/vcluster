package pro

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/pkg/options"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func ValidateProOptions(options *options.VirtualClusterOptions) error {
	_, _, err := ParseExtraHooks(options.ProOptions.EnforceValidatingHooks, options.ProOptions.EnforceMutatingHooks)
	return err
}

func ParseExtraHooks(valHooks, mutHooks []string) ([]admissionregistrationv1.ValidatingWebhookConfiguration, []admissionregistrationv1.MutatingWebhookConfiguration, error) {
	decodedVal := make([]string, 0, len(valHooks))
	for _, v := range valHooks {
		bytes, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, nil, err
		}
		decodedVal = append(decodedVal, string(bytes))
	}
	decodedMut := make([]string, 0, len(mutHooks))
	for _, v := range mutHooks {
		bytes, err := base64.StdEncoding.DecodeString(v)
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
