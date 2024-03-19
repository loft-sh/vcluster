package config

import (
	"fmt"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
)

func validateMappings(resolveDNS []vclusterconfig.ResolveDNS) error {
	for i, mapping := range resolveDNS {
		// parse service format
		options := 0
		if mapping.Service != "" {
			options++
			if strings.Count(mapping.Service, "/") != 1 {
				return fmt.Errorf("error validating networking.resolveDNS[%d].service: expected format namespace/name, but got %s", i, mapping.Service)
			}
		}
		if mapping.Hostname != "" {
			options++
		}
		if mapping.Namespace != "" {
			if mapping.Target.HostNamespace == "" {
				return fmt.Errorf("error validating networking.resolveDNS[%d].namespace: when using namespace, target.hostNamespace is required", i)
			}

			options++
		} else if mapping.Target.HostNamespace != "" {
			return fmt.Errorf("error validating networking.resolveDNS[%d]: when using target.hostNamespace, .namespace is required", i)
		}

		if options == 0 {
			return fmt.Errorf("at least one option required for networking.resolveDNS[%d]", i)
		} else if options > 1 {
			return fmt.Errorf("only a single option allowed for networking.resolveDNS[%d]", i)
		}

		// validate targets
		err := validateTarget(mapping.Target)
		if err != nil {
			return fmt.Errorf("error validating networking.resolveDNS[%d].to", i)
		}
	}

	return nil
}

func validateTarget(target vclusterconfig.ResolveDNSTarget) error {
	options := 0
	if target.Hostname != "" {
		options++
	}
	if target.IP != "" {
		options++
	}
	if target.HostNamespace != "" {
		options++
	}
	if target.HostService != "" {
		options++

		// check if service is defined with the namespace/name format
		if strings.Count(target.HostService, "/") != 1 {
			return fmt.Errorf("expected namespace/name format for .to.service, but got %s", target.HostService)
		}
	}
	if target.VClusterService != "" {
		options++

		// check if vcluster service is defined with namespace/name format
		if strings.Count(target.VClusterService, "/") != 3 {
			return fmt.Errorf("expected hostNamespace/vClusterName/vClusterNamespace/vClusterService format for .to.vClusterService, but got %s", target.VClusterService)
		}
	}
	if options == 0 {
		return fmt.Errorf("at least one option required for .to")
	} else if options > 1 {
		return fmt.Errorf("only a single option allowed for .to")
	}

	return nil
}
