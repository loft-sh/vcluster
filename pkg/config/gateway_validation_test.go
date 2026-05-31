package config

import (
	"strings"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
)

func TestGatewayFromHostDefaults(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.HostNamespaces = []string{"networking"}

	if err := ValidateConfigAndSetDefaults(vcConfig); err != nil {
		t.Fatalf("ValidateConfigAndSetDefaults() error = %v", err)
	}
	if vcConfig.Sync.FromHost.Gateways.VirtualNamespace != "vcluster-gateways" {
		t.Fatalf("expected default virtual namespace vcluster-gateways, got %q", vcConfig.Sync.FromHost.Gateways.VirtualNamespace)
	}
	if !vcConfig.Sync.FromHost.Gateways.Sanitize.CertificateRefs {
		t.Fatalf("expected certificate refs to be sanitized by default")
	}
	if !vcConfig.Sync.FromHost.Gateways.Sanitize.Infrastructure {
		t.Fatalf("expected infrastructure to be sanitized by default")
	}
	if vcConfig.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled != "auto" {
		t.Fatalf("expected referenceGrants.enabled default auto, got %q", vcConfig.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled)
	}
}

func TestGatewayFromHostValidationRejectsSelectorWithoutHostNamespaces(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Selector.MatchLabels = map[string]string{"shared": "true"}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "sync.fromHost.gateways.hostNamespaces") {
		t.Fatalf("expected hostNamespaces validation error, got %v", err)
	}
}

func TestGatewayFromHostValidationRejectsInvalidImports(t *testing.T) {
	for _, tt := range []struct {
		name    string
		imports []rootconfig.GatewayImport
		want    string
	}{
		{
			name:    "missing host namespace",
			imports: []rootconfig.GatewayImport{{Name: "edge"}},
			want:    "hostNamespace",
		},
		{
			name:    "missing name",
			imports: []rootconfig.GatewayImport{{HostNamespace: "networking"}},
			want:    "name",
		},
		{
			name: "duplicate virtual name",
			imports: []rootconfig.GatewayImport{
				{HostNamespace: "networking", Name: "edge", VirtualName: "shared"},
				{HostNamespace: "platform", Name: "edge", VirtualName: "shared"},
			},
			want: "duplicate virtualName",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			vcConfig := &VirtualClusterConfig{}
			vcConfig.Sync.FromHost.Gateways.Enabled = true
			vcConfig.Sync.FromHost.Gateways.Imports = tt.imports
			err := ValidateConfigAndSetDefaults(vcConfig)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
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
	vcConfig.Sync.FromHost.Gateways.HostNamespaces = []string{"networking"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy = &rootconfig.GatewayVirtualNamespacePolicy{From: "Anywhere"}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "allowedRoutes.defaultVirtualNamespacePolicy.from") {
		t.Fatalf("expected allowedRoutes policy validation error, got %v", err)
	}
}

func TestGatewayImportValidationRejectsInvalidHostnamePattern(t *testing.T) {
	vcConfig := &VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Imports = []rootconfig.GatewayImport{{
		HostNamespace:    "networking",
		Name:             "edge",
		AllowedHostnames: []string{"bad_*_hostname"},
	}}

	err := ValidateConfigAndSetDefaults(vcConfig)
	if err == nil || !strings.Contains(err.Error(), "allowedHostnames") {
		t.Fatalf("expected hostname validation error, got %v", err)
	}
}
