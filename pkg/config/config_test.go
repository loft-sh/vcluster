package config

import (
	"testing"
)

func TestConfigParsing(t *testing.T) {
	rawConfig := `version: v1beta1
export:  # Old from virtual cluster
- apiVersion: cert-manager.io/v1
  kind: Issuer
- apiVersion: cert-manager.io/v1
  kind: Certificates
  patches:
    - op: rewriteName
      path: spec.ca.secretName
  reversePatches:       
    - op: copyFromObject # Sync status back by default
      fromPath: status
      path: status`

	_, err := Parse(rawConfig)
	if err != nil {
		t.Fatalf("Error parsing config %v", err)
	}

}
