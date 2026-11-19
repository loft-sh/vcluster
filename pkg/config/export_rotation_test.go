package config

import (
	"testing"

	"github.com/loft-sh/vcluster/config"
	"k8s.io/utils/ptr"
)

func TestValidateExportServiceAccountLifetime(t *testing.T) {
	for _, test := range []struct {
		name     string
		account  config.ExportKubeConfigServiceAccount
		wantFail bool
	}{
		{name: "legacy"},
		{name: "empty unnamed rotation block", account: config.ExportKubeConfigServiceAccount{TokenRotation: &config.ExportKubeConfigTokenRotation{}}},
		{name: "percentage without expiration", account: rotationAccount("", ptr.To[int32](33)), wantFail: true},
		{name: "legacy ServiceAccount", account: config.ExportKubeConfigServiceAccount{Name: "runtime"}},
		{name: "short-lived", account: rotationAccount("10m", nil)},
		{name: "seconds", account: rotationAccount("600s", nil)},
		{name: "compound duration", account: rotationAccount("1h30m", nil)},
		{name: "fractional hours", account: rotationAccount("0.5h", nil)},
		{name: "milliseconds", account: rotationAccount("600000ms", nil)},
		{name: "API maximum", account: rotationAccount("4294967296s", nil)},
		{name: "above API maximum", account: rotationAccount("4294967297s", nil), wantFail: true},
		{name: "negative", account: rotationAccount("-1s", nil), wantFail: true},
		{name: "zero", account: rotationAccount("0s", nil), wantFail: true},
		{name: "too short", account: rotationAccount("599s", nil), wantFail: true},
		{name: "fractional seconds", account: rotationAccount("600.5s", nil), wantFail: true},
		{name: "subsecond remainder", account: rotationAccount("10m1ns", nil), wantFail: true},
		{name: "overflow", account: rotationAccount("999999999999999999999h", nil), wantFail: true},
		{name: "missing unit", account: rotationAccount("600", nil), wantFail: true},
		{name: "malformed", account: rotationAccount("ten minutes", nil), wantFail: true},
		{name: "unnamed", account: config.ExportKubeConfigServiceAccount{TokenRotation: &config.ExportKubeConfigTokenRotation{Expiration: "10m"}}, wantFail: true},
		{name: "early renewal", account: rotationAccount("10m", ptr.To[int32](80))},
		{name: "late renewal", account: rotationAccount("10m", ptr.To[int32](1))},
		{name: "zero percentage", account: rotationAccount("10m", ptr.To[int32](0)), wantFail: true},
		{name: "negative percentage", account: rotationAccount("10m", ptr.To[int32](-1)), wantFail: true},
		{name: "full percentage", account: rotationAccount("10m", ptr.To[int32](100)), wantFail: true},
		{name: "empty rotation block", account: rotationAccount("", nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, options := range []config.ExportKubeConfig{
				{ExportKubeConfigProperties: config.ExportKubeConfigProperties{ServiceAccount: test.account}},
				{AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{{
					Name: "export", ExportKubeConfigProperties: config.ExportKubeConfigProperties{ServiceAccount: test.account},
				}}},
			} {
				err := validateExportKubeConfig(options)
				if (err != nil) != test.wantFail {
					t.Fatalf("validate export: got %v, want failure %v", err, test.wantFail)
				}
			}
		})
	}
}

func rotationAccount(expiration config.Duration, percentage *int32) config.ExportKubeConfigServiceAccount {
	return config.ExportKubeConfigServiceAccount{
		Name: "runtime",
		TokenRotation: &config.ExportKubeConfigTokenRotation{
			Expiration:            expiration,
			RenewBeforePercentage: percentage,
		},
	}
}
