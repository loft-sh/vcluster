package config

import (
	"strings"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
)

func TestGatewayFromHostValidationAcceptsMappingShape(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{
		"platform-gateways/*": "platform-gateways/*",
		"networking/edge":     "shared-gateways/edge-public",
	}

	if err := ValidateConfigAndSetDefaults(vcConfig); err != nil {
		t.Fatalf("ValidateConfigAndSetDefaults() error = %v", err)
	}
	if vcConfig.Sync.FromHost.Gateways.Sanitize.CertificateRefs {
		t.Fatalf("validation must not force certificate ref sanitization on when explicitly disabled")
	}
	if vcConfig.Sync.FromHost.Gateways.Sanitize.Infrastructure {
		t.Fatalf("validation must not force infrastructure sanitization on when explicitly disabled")
	}
	if vcConfig.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled != "auto" {
		t.Fatalf("expected referenceGrants.enabled default auto, got %q", vcConfig.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled)
	}
}

func TestGatewayFromHostValidationRequiresMappingsWhenEnabled(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "sync.fromHost.gateways.mappings") {
		t.Fatalf("expected mappings validation error, got %v", err)
	}
}

func TestGatewayFromHostValidationRejectsInvalidMappings(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mappings map[string]string
		want     string
	}{
		{
			name:     "empty host namespace wildcard",
			mappings: map[string]string{"": "tenant-gateways/*"},
			want:     "explicit host namespace",
		},
		{
			name:     "empty host namespace exact",
			mappings: map[string]string{"/edge": "tenant-gateways/edge"},
			want:     "explicit host namespace",
		},
		{
			name:     "wildcard value does not preserve name",
			mappings: map[string]string{"platform/*": "tenant-gateways/edge"},
			want:     "must map to",
		},
		{
			name: "overlapping source wildcard and exact",
			mappings: map[string]string{
				"platform/*":    "tenant-gateways/*",
				"platform/edge": "shared-gateways/edge",
			},
			want: "overlaps",
		},
		{
			name: "duplicate exact target",
			mappings: map[string]string{
				"platform/edge":   "shared-gateways/edge",
				"networking/edge": "shared-gateways/edge",
			},
			want: "duplicate",
		},
		{
			name: "exact target inside wildcard target namespace",
			mappings: map[string]string{
				"platform/*":      "shared-gateways/*",
				"networking/edge": "shared-gateways/network-edge",
			},
			want: "wildcard",
		},
		{
			name: "duplicate wildcard target namespace",
			mappings: map[string]string{
				"platform/*":   "shared-gateways/*",
				"networking/*": "shared-gateways/*",
			},
			want: "duplicate",
		},
		{
			name:     "system tenant namespace",
			mappings: map[string]string{"platform/edge": "kube-system/edge"},
			want:     "reserved",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			vcConfig := &VirtualClusterConfig{}
			vcConfig.Sync.FromHost.Gateways.Enabled = true
			vcConfig.Sync.FromHost.Gateways.Mappings.ByName = tt.mappings
			err := ValidateConfigAndSetDefaults(vcConfig)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestGatewayFromHostValidationRejectsUnmappedAllowedRoutesOverride(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"platform/edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace: "platform",
		Name:          "internal",
		VirtualNamespacePolicy: rootconfig.GatewayVirtualNamespacePolicy{
			From: "Same",
		},
	}}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "not covered") {
		t.Fatalf("expected unmapped override validation error, got %v", err)
	}
}

func TestGatewayReferenceGrantModeValidation(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "sometimes"

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "sync.toHost.gatewayApi.referenceGrants.enabled") {
		t.Fatalf("expected referenceGrants mode validation error, got %v", err)
	}
}

func TestGatewayFromHostValidationRejectsInvalidAllowedRoutesPolicy(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy = &rootconfig.GatewayVirtualNamespacePolicy{From: "Anywhere"}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "allowedRoutes.defaultVirtualNamespacePolicy.from") {
		t.Fatalf("expected allowedRoutes policy validation error, got %v", err)
	}
}

func TestGatewayAllowedRoutesOverrideRejectsInvalidHostnamePattern(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "edge",
		AllowedHostnames: []string{"bad_*_hostname"},
	}}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "allowedHostnames") {
		t.Fatalf("expected hostname validation error, got %v", err)
	}
}
