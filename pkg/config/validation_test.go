package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/util/namespaces"
)

func Test(t *testing.T) {
	validURL := "https://loft.sh"
	validCABUNDLE := `-----BEGIN CERTIFICATE-----
MIICFTCCAZugAwIBAgIQPZg7pmY9kGP3fiZXOATvADAKBggqhkjOPQQDAzBMMS4wLAYDVQQDDCVB
dG9zIFRydXN0ZWRSb290IFJvb3QgQ0EgRUNDIFRMUyAyMDIxMQ0wCwYDVQQKDARBdG9zMQswCQYD
VQQGEwJERTAeFw0yMTA0MjIwOTI2MjNaFw00MTA0MTcwOTI2MjJaMEwxLjAsBgNVBAMMJUF0b3Mg
VHJ1c3RlZFJvb3QgUm9vdCBDQSBFQ0MgVExTIDIwMjExDTALBgNVBAoMBEF0b3MxCzAJBgNVBAYT
AkRFMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEloZYKDcKZ9Cg3iQZGeHkBQcfl+3oZIK59sRxUM6K
DP/XtXa7oWyTbIOiaG6l2b4siJVBzV3dscqDY4PMwL502eCdpO5KTlbgmClBk1IQ1SQ4AjJn8ZQS
b+/Xxd4u/RmAo0IwQDAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBR2KCXWfeBmmnoJsmo7jjPX
NtNPojAOBgNVHQ8BAf8EBAMCAYYwCgYIKoZIzj0EAwMDaAAwZQIwW5kp85wxtolrbNa9d+F851F+
uDrNozZffPc8dz7kUK2o59JZDCaOMDtuCCrCp1rIAjEAmeMM56PDr9NJLkaCI2ZdyQAUEv049OGY
a3cpetskz2VAv9LcjBHo9H1/IISpQuQo
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFZDCCA0ygAwIBAgIQU9XP5hmTC/srBRLYwiqipDANBgkqhkiG9w0BAQwFADBMMS4wLAYDVQQD
DCVBdG9zIFRydXN0ZWRSb290IFJvb3QgQ0EgUlNBIFRMUyAyMDIxMQ0wCwYDVQQKDARBdG9zMQsw
CQYDVQQGEwJERTAeFw0yMTA0MjIwOTIxMTBaFw00MTA0MTcwOTIxMDlaMEwxLjAsBgNVBAMMJUF0
b3MgVHJ1c3RlZFJvb3QgUm9vdCBDQSBSU0EgVExTIDIwMjExDTALBgNVBAoMBEF0b3MxCzAJBgNV
BAYTAkRFMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAtoAOxHm9BYx9sKOdTSJNy/BB
l01Z4NH+VoyX8te9j2y3I49f1cTYQcvyAh5x5en2XssIKl4w8i1mx4QbZFc4nXUtVsYvYe+W/CBG
vevUez8/fEc4BKkbqlLfEzfTFRVOvV98r61jx3ncCHvVoOX3W3WsgFWZkmGbzSoXfduP9LVq6hdK
ZChmFSlsAvFr1bqjM9xaZ6cF4r9lthawEO3NUDPJcFDsGY6wx/J0W2tExn2WuZgIWWbeKQGb9Cpt
0xU6kGpn8bRrZtkh68rZYnxGEFzedUlnnkL5/nWpo63/dgpnQOPF943HhZpZnmKaau1Fh5hnstVK
PNe0OwANwI8f4UDErmwh3El+fsqyjW22v5MvoVw+j8rtgI5Y4dtXz4U2OLJxpAmMkokIiEjxQGMY
sluMWuPD0xeqqxmjLBvk1cbiZnrXghmmOxYsL3GHX0WelXOTwkKBIROW1527k2gV+p2kHYzygeBY
Br3JtuP2iV2J+axEoctr+hbxx1A9JNr3w+SH1VbxT5Aw+kUJWdo0zuATHAR8ANSbhqRAvNncTFd+
rrcztl524WWLZt+NyteYr842mIycg5kDcPOvdO3GDjbnvezBc6eUWsuSZIKmAMFwoW4sKeFYV+xa
fJlrJaSQOoD0IJ2azsct+bJLKZWD6TWNp0lIpw9MGZHQ9b8Q4HECAwEAAaNCMEAwDwYDVR0TAQH/
BAUwAwEB/zAdBgNVHQ4EFgQUdEmZ0f+0emhFdcN+tNzMzjkz2ggwDgYDVR0PAQH/BAQDAgGGMA0G
CSqGSIb3DQEBDAUAA4ICAQAjQ1MkYlxt/T7Cz1UAbMVWiLkO3TriJQ2VSpfKgInuKs1l+NsW4AmS
4BjHeJi78+xCUvuppILXTdiK/ORO/auQxDh1MoSf/7OwKwIzNsAQkG8dnK/haZPso0UvFJ/1TCpl
Q3IM98P4lYsU84UgYt1UU90s3BiVaU+DR3BAM1h3Egyi61IxHkzJqM7F78PRreBrAwA0JrRUITWX
AdxfG/F851X6LWh3e9NpzNMOa7pNdkTWwhWaJuywxfW70Xp0wmzNxbVe9kzmWy2B27O3Opee7c9G
slA9hGCZcbUztVdF5kJHdWoOsAgMrr3e97sPWD2PAzHoPYJQyi9eDF20l74gNAf0xBLh7tew2Vkt
afcxBPTy+av5EzH4AXcOPUIjJsyacmdRIXrMPIWo6iFqO9taPKU0nprALN+AnCng33eU0aKAQv9q
TFsR0PXNor6uzFFcw9VUewyu1rkGd4Di7wcaaMxZUa1+XGdrudviB0JbuAEFWDlN5LuYo7Ey7Nmj
1m+UI/87tyll5gfp77YZ6ufCOB0yiJA8EytuzO+rdwY0d4RPcuSBhPm5dDTedk+SKlOxJTnbPP/l
PqYO5Wue/9vsL3SD3460s6neFE3/MaNFcyT6lSnMEpcEoji2jbDwN/zIIX8/syQbPYtuzE2wFg2W
HYMfRsCbvUOZ58SWLs5fyQ==
-----END CERTIFICATE-----`
	testCases := []struct {
		name    string
		wantErr string
		valHook []config.ValidatingWebhookConfiguration
		mutHook []config.MutatingWebhookConfiguration
	}{
		{
			name:    "valid valhook",
			valHook: []config.ValidatingWebhookConfiguration{valHook(config.ValidatingWebhookClientConfig{Service: &config.ValidatingWebhookServiceReference{Namespace: "test", Name: "service"}})},
		},
		{
			name:    "valid muthook",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{Service: &config.ValidatingWebhookServiceReference{Namespace: "test", Name: "service"}})},
		},
		{
			name:    "invalid valhook",
			valHook: []config.ValidatingWebhookConfiguration{valHook(config.ValidatingWebhookClientConfig{})},
			wantErr: "webhook client config was not valid for ValidatingWebhookConfiguration test: there is no service config",
		},
		{
			name:    "invalid muthook",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{})},
			wantErr: "webhook client config was not valid for MutatingWebhookConfiguration test: there is no service config",
		},
		{
			name:    "invalid service",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{Service: &config.ValidatingWebhookServiceReference{Namespace: "test"}})},
			wantErr: "webhook client config was not valid for MutatingWebhookConfiguration test: namespace or name of the service is missing",
		},
		{
			name:    "valid url",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{URL: &validURL})},
		},
		{
			name:    "invalid bundle",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{Service: &config.ValidatingWebhookServiceReference{Namespace: "test"}, CABundle: []byte("HAZAA")})},
			wantErr: "webhook client config was not valid for MutatingWebhookConfiguration test: could not parse the CABundle",
		},
		{
			name:    "valid bundle",
			mutHook: []config.MutatingWebhookConfiguration{mutHook(config.ValidatingWebhookClientConfig{Service: &config.ValidatingWebhookServiceReference{Namespace: "test", Name: "test"}, CABundle: []byte(validCABUNDLE)})},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseExtraHooks(tt.valHook, tt.mutHook)
			if err != nil && (tt.wantErr == "" || tt.wantErr != err.Error()) {
				t.Errorf("wanted err to be %s but got %s", tt.wantErr, err.Error())
			} else if err == nil && tt.wantErr != "" {
				t.Errorf("wanted err to be %s but got nil", tt.wantErr)
			}
		})
	}
}

func valHook(clientCfg config.ValidatingWebhookClientConfig) config.ValidatingWebhookConfiguration {
	hook := config.ValidatingWebhookConfiguration{}
	hook.APIVersion = "v1"
	hook.Kind = "ValidatingWebhookConfiguration"
	hook.Metadata.Name = "test"
	hook.Webhooks = []config.ValidatingWebhook{
		{Name: "test", ClientConfig: clientCfg},
	}
	return hook
}

func mutHook(clientCfg config.ValidatingWebhookClientConfig) config.MutatingWebhookConfiguration {
	hook := config.MutatingWebhookConfiguration{}
	hook.APIVersion = "v1"
	hook.Kind = "MutatingWebhookConfiguration"
	hook.Metadata.Name = "test"
	hook.Webhooks = []config.MutatingWebhook{
		{
			ValidatingWebhook: config.ValidatingWebhook{Name: "test", ClientConfig: clientCfg},
		},
	}
	return hook
}

func TestValidateFromHostSyncMappings(t *testing.T) {
	noErr := func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("expected err to be nil but got %v", err)
		}
	}
	expectErr := func(t *testing.T, err error) {
		if err == nil {
			t.Errorf("expected error got nil")
		}
	}
	cases := []struct {
		name      string
		cmConfig  config.EnableSwitchWithResourcesMappings
		expectErr func(t *testing.T, err error)
	}{
		{
			name: "valid config",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"from-host-sync-test/*": "barfoo/*",
						"default/my-cm":         "barfoo/cm-my",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config 2",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"":              "barfoo/*",
						"default/my-cm": "barfoo/cm-my",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config 3",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"":              "barfoo",
						"default/my-cm": "barfoo/cm-my",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config 4",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"/my-cm":        "barfoo/my-cm",
						"default/my-cm": "barfoo/cm-my",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config 5",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"": "barfoo/",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config: dot in the object name",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"my-ns/my-cm": "barfoor/my.cm",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config: dot in the host object name",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"my-ns/my.cm": "barfoor/my-cm",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "valid config: dots in object names",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"my-ns/my.cm": "barfoor/my.cm",
					},
				},
			},
			expectErr: noErr,
		},
		{
			name: "(invalid) host namespace mapped to object",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default": "barfoo/cm-my",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) host object mapped to namespace",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/my-cm": "barfoo",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) wildcard used in host but not in virtual",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/*": "barfoo",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) '*' is not valid key",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/my-cm": "barfoo/my-cm",
						"*":             "barfoo2/*",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) host object name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/_not_valid_obj_name": "barfoo/my-cm",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) host namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"_not-Valid_namespace_name/my-cm": "barfoo/my-cm",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) virtual object name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/my-cm": "barfoo/_not_valid_obj_name",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) map from host object, but virtual namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"default/my-cm": "_not_valid_ns66_name/my-cm",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) map from host vcluster namespace, but virtual namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"/my-cm": "_not_valid_ns66_name/my-cm",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) host name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"/_not_valid_obj_name": "default/my-cm",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) virtual namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"": "!66_not_valid_ns/*",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) virtual namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"my.ns/host-obj": "valid/valid",
					},
				},
			},
			expectErr: expectErr,
		},
		{
			name: "(invalid) virtual namespace name is not valid DNS1123Label",
			cmConfig: config.EnableSwitchWithResourcesMappings{
				Enabled: true,
				Mappings: config.FromHostMappings{
					ByName: map[string]string{
						"in.valid/host-obj": "in.valid/valid",
					},
				},
			},
			expectErr: expectErr,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFromHostSyncMappings(tc.cmConfig, "configMaps")
			tc.expectErr(t, err)
		})
	}
}

func TestValidateFromHostSyncCustomResources(t *testing.T) {
	cases := []struct {
		name                        string
		customResourcesFromHostSync map[string]config.SyncFromHostCustomResource
		checkErr                    func(t *testing.T, err error)
	}{
		{
			name: "valid cluster scope crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"clusterissuers.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeCluster,
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "invalid cluster scope crd config with mappings",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"clusterissuers.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeCluster,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"": "default",
						},
					},
				},
			},
			checkErr: expectErr(".selector.mappings are only supported for sync.fromHost.customResources['clusterissuers.cert-manager.io'] with scope 'Namespaced'"),
		},
		{
			name: "valid namespaced scope crd config with mappings",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"": "default",
						},
					},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "invalid namespaced scope crd config without mappings",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			checkErr: expectErr(".selector.mappings is required for Namespaced scope sync.fromHost.customResources['certificaterequests.cert-manager.io']"),
		},
		{
			name: "invalid scope for crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   "dummy",
				},
			},
			checkErr: expectErr("unsupported scope dummy for sync.fromHost.customResources['certificaterequests.cert-manager.io'].scope. Only 'Cluster' and 'Namespaced' are allowed"),
		},
		{
			name: "invalid namespace for crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"_s66/my": "default/my-cr",
						},
					},
				},
			},
			checkErr: expectErr("config.sync.fromHost.certificaterequests.cert-manager.io.selector.mappings parsed host namespace is not valid namespace name [a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')]"),
		},
		{
			name: "invalid name for crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"/_s66": "default/my-cr",
						},
					},
				},
			},
			checkErr: expectErr("config.sync.fromHost.certificaterequests.cert-manager.io.selector.mappings parsed object name from key (/_s66) is not valid name [a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')]"),
		},
		{
			name: "invalid virtual namespace for crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"default/my-cr": "_s66/my",
						},
					},
				},
			},
			checkErr: expectErr("config.sync.fromHost.certificaterequests.cert-manager.io.selector.mappings parsed virtual namespace is not valid namespace name [a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')]"),
		},
		{
			name: "invalid virtual name for crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"default/my-cr": "/_s66",
						},
					},
				},
			},
			checkErr: expectErr("config.sync.fromHost.certificaterequests.cert-manager.io.selector.mappings parsed virtual namespace is not valid namespace name [a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')]"),
		},
		{
			name: "invalid virtual entry in crd config",
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"certificaterequests.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
					Mappings: config.FromHostMappings{
						ByName: map[string]string{
							"default/my-cr": "default/my/s66",
						},
					},
				},
			},
			checkErr: expectErr("config.sync.fromHost.certificaterequests.cert-manager.io.selector.mappings has key:value pair in invalid format: default/my-cr:default/my/s66 (expected NAMESPACE_NAME/NAME, NAMESPACE_NAME/*, /NAME or \"\")"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFromHostSyncCustomResources(tc.customResourcesFromHostSync)
			tc.checkErr(t, err)
		})
	}
}

func TestValidateExportKubeConfig(t *testing.T) {
	cases := []struct {
		name             string
		exportKubeConfig config.ExportKubeConfig
		expectedError    error
	}{
		{
			name: "Setting only exportKubeConfig.secret is valid",
			exportKubeConfig: config.ExportKubeConfig{
				Secret: config.ExportKubeConfigSecretReference{
					Name: "my-secret",
				},
			},
		},
		{
			name: "Setting only exportKubeConfig.additionalSecrets with name is valid",
			exportKubeConfig: config.ExportKubeConfig{
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{
						Name: "my-secret",
					},
				},
			},
		},
		{
			name: "Setting only exportKubeConfig.additionalSecrets with namespace is valid",
			exportKubeConfig: config.ExportKubeConfig{
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{
						Namespace: "my-namespace",
					},
				},
			},
		},
		{
			name: "Setting both exportKubeConfig.secret and exportKubeConfig.additionalSecrets is not valid",
			exportKubeConfig: config.ExportKubeConfig{
				Secret: config.ExportKubeConfigSecretReference{
					Name: "my-secret-1",
				},
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{
						Name: "my-secret-2",
					},
				},
			},
			expectedError: errExportKubeConfigBothSecretAndAdditionalSecretsSet,
		},
		{
			name: "Setting empty additional secret is not valid",
			exportKubeConfig: config.ExportKubeConfig{
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{},
				},
			},
			expectedError: errExportKubeConfigAdditionalSecretWithoutNameAndNamespace,
		},
		{
			name: "Setting non-empty additional secret, but without Name and Namespace, is not valid",
			exportKubeConfig: config.ExportKubeConfig{
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{
						ExportKubeConfigProperties: config.ExportKubeConfigProperties{
							Context: "my-context",
							Server:  "my-server",
						},
					},
				},
			},
			expectedError: errExportKubeConfigAdditionalSecretWithoutNameAndNamespace,
		},
		{
			name: "Setting only exportKubeConfig.server is invalid",
			exportKubeConfig: config.ExportKubeConfig{
				ExportKubeConfigProperties: config.ExportKubeConfigProperties{
					Server: "my-server",
				},
			},
			expectedError: errExportKubeConfigServerNotValid,
		},
		{
			name: "Setting only exportKubeConfig.server is valid with https://",
			exportKubeConfig: config.ExportKubeConfig{
				ExportKubeConfigProperties: config.ExportKubeConfigProperties{
					Server: "https://my-server.com",
				},
			},
		},
		{
			name: "Setting only exportKubeConfig.server is valid with https:// and port",
			exportKubeConfig: config.ExportKubeConfig{
				ExportKubeConfigProperties: config.ExportKubeConfigProperties{
					Server: "https://my-server.com:443",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actualError := validateExportKubeConfig(tc.exportKubeConfig)
			if tc.expectedError == nil && actualError != nil {
				t.Errorf("expected validation to pass, but got error: %v", actualError)
			}
			if tc.expectedError != nil {
				if actualError == nil {
					t.Errorf("expected validation to fail with error %q, but it passed", tc.expectedError)
				} else if !errors.Is(actualError, tc.expectedError) {
					t.Errorf("expected to get error %q, but instead got other error: %v", tc.expectedError, actualError)
				}
			}
		})
	}
}

const patchesPath = "spec.containers[*].name"

func TestValidateAllSyncPatches(t *testing.T) {
	cases := []struct {
		name          string
		configSync    config.Sync
		expectedError error
	}{
		{
			name: "Simple config sync with one patch",
			configSync: config.Sync{
				ToHost: config.SyncToHost{
					Pods: config.SyncPods{
						Patches: []config.TranslatePatch{
							{
								Path:              patchesPath,
								Expression:        "\"my-prefix-\"+value",
								ReverseExpression: "value.slice(\"my-prefix\".length)",
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple patches with same path",
			configSync: config.Sync{
				ToHost: config.SyncToHost{
					Pods: config.SyncPods{
						Patches: []config.TranslatePatch{
							{
								Path:              patchesPath,
								Expression:        "\"my-prefix-\"+value",
								ReverseExpression: "value.slice(\"my-prefix\".length)",
							},
						},
					},
					Endpoints: config.EnableSwitchWithPatches{
						Patches: []config.TranslatePatch{
							{
								Path:              patchesPath,
								Expression:        "\"my-prefix-\"+value",
								ReverseExpression: "value.slice(\"my-prefix\".length)",
							},
						},
					},
				},
			},
		},
		{
			name: "Invalid duplicated paths on patches",
			configSync: config.Sync{
				ToHost: config.SyncToHost{
					Pods: config.SyncPods{
						Patches: []config.TranslatePatch{
							{
								Path:       patchesPath,
								Expression: "\"my-prefix-\"+value",
							},
							{
								Path:              patchesPath,
								ReverseExpression: "value.slice(\"my-prefix\".length)",
							},
						},
					},
				},
			},
			expectedError: errors.New("sync.toHost.pods.patches[0] and sync.toHost.pods.patches[1] have the same path \"spec.containers[*].name\""),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAllSyncPatches(tc.configSync)
			if tc.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, but got nil", tc.expectedError)
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("expected error %v, but got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, but got %v", err)
				}
			}
		})
	}
}

func TestValidateToHostSyncAndIstioIntegration(t *testing.T) {
	istioEnabled := config.Istio{
		EnableSwitch: config.EnableSwitch{Enabled: true},
		Sync: config.IstioSync{ToHost: config.IstioSyncToHost{
			DestinationRules: config.EnableSwitch{Enabled: true},
			Gateways:         config.EnableSwitch{Enabled: true},
			VirtualServices:  config.EnableSwitch{Enabled: true},
		}},
	}
	istioDisabled := config.Istio{
		EnableSwitch: config.EnableSwitch{Enabled: false},
	}

	cases := []struct {
		name                      string
		customResourcesToHostSync map[string]config.SyncToHostCustomResource
		istioIntegration          config.Istio
		checkErr                  func(t *testing.T, err error)
	}{
		{
			name: "valid config",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"clusterissuers.cert-manager.io": {
					Enabled: true,
					Scope:   config.ScopeCluster,
				},
			},
			istioIntegration: istioEnabled,
			checkErr:         noErrExpected,
		},
		{
			name: "destination rule listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"destinationrules.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioEnabled,
			checkErr: expectErr("istio integration is enabled but istio custom resource (destinationrules.networking.istio.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "destination rule listed in sync.toHost.customResources but istio disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"destinationrules.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioDisabled,
			checkErr:         noErrExpected,
		},
		{
			name: "gateways listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"gateways.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioEnabled,
			checkErr: expectErr("istio integration is enabled but istio custom resource " +
				"(gateways.networking.istio.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "gateways listed in sync.toHost.customResources but istio disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"gateways.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioDisabled,
			checkErr:         noErrExpected,
		},
		{
			name: "virtual services listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"virtualservices.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioEnabled,
			checkErr: expectErr("istio integration is enabled but istio custom resource " +
				"(virtualservices.networking.istio.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "virtual services listed in sync.toHost.customResources but istio disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"virtualservices.networking.istio.io": {
					Enabled: true,
					Scope:   config.ScopeNamespaced,
				},
			},
			istioIntegration: istioDisabled,
			checkErr:         noErrExpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIstioEnabled(tc.customResourcesToHostSync, tc.istioIntegration)
			tc.checkErr(t, err)
		})
	}
}

func TestValidateToHostSyncAndCertManagerIntegration(t *testing.T) {
	certManagerEnabled := config.CertManager{
		EnableSwitch: config.EnableSwitch{Enabled: true},
		Sync: config.CertManagerSync{
			ToHost: config.CertManagerSyncToHost{
				Certificates: config.EnableSwitch{Enabled: true},
				Issuers:      config.EnableSwitch{Enabled: true},
			},
			FromHost: config.CertManagerSyncFromHost{
				ClusterIssuers: config.ClusterIssuersSyncConfig{
					EnableSwitch: config.EnableSwitch{Enabled: true},
				},
			},
		},
	}
	certManagerDisabled := config.CertManager{
		EnableSwitch: config.EnableSwitch{Enabled: false},
	}

	cases := []struct {
		name                        string
		customResourcesToHostSync   map[string]config.SyncToHostCustomResource
		customResourcesFromHostSync map[string]config.SyncFromHostCustomResource
		certManagerIntegration      config.CertManager
		checkErr                    func(t *testing.T, err error)
	}{
		{
			name: "valid config",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"some.other.resource": {Enabled: true},
			},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerEnabled,
			checkErr:                    noErrExpected,
		},
		{
			name: "certificates listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"certificates.cert-manager.io": {Enabled: true},
			},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerEnabled,
			checkErr: expectErr("cert-manager integration is enabled but cert-manager custom resource " +
				"(certificates.cert-manager.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "issuers listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"issuers.cert-manager.io": {Enabled: true},
			},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerEnabled,
			checkErr: expectErr("cert-manager integration is enabled but cert-manager custom resource " +
				"(issuers.cert-manager.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name:                      "clusterissuers listed in sync.fromHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"clusterissuers.cert-manager.io": {Enabled: true},
			},
			certManagerIntegration: certManagerEnabled,
			checkErr: expectErr("cert-manager integration is enabled but cert-manager custom resource " +
				"(clusterissuers.cert-manager.io) is also set in the sync.fromHost.customResources. " +
				"This is not supported, please remove the entry from sync.fromHost.customResources"),
		},
		{
			name: "certificates listed in sync.toHost.customResources but cert-manager disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"certificates.cert-manager.io": {Enabled: true},
			},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerDisabled,
			checkErr:                    noErrExpected,
		},
		{
			name: "issuers listed in sync.toHost.customResources but cert-manager disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"issuers.cert-manager.io": {Enabled: true},
			},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerDisabled,
			checkErr:                    noErrExpected,
		},
		{
			name:                      "clusterissuers listed in sync.fromHost.customResources but cert-manager disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{
				"clusterissuers.cert-manager.io": {Enabled: true},
			},
			certManagerIntegration: certManagerDisabled,
			checkErr:               noErrExpected,
		},
		{
			name:                        "no custom resources and cert-manager enabled",
			customResourcesToHostSync:   map[string]config.SyncToHostCustomResource{},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerEnabled,
			checkErr:                    noErrExpected,
		},
		{
			name:                        "no custom resources and cert-manager disabled",
			customResourcesToHostSync:   map[string]config.SyncToHostCustomResource{},
			customResourcesFromHostSync: map[string]config.SyncFromHostCustomResource{},
			certManagerIntegration:      certManagerDisabled,
			checkErr:                    noErrExpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCertManagerEnabled(tc.customResourcesToHostSync, tc.customResourcesFromHostSync, tc.certManagerIntegration)
			tc.checkErr(t, err)
		})
	}
}

func TestValidateToHostSyncAndKubeVirtIntegration(t *testing.T) {
	kubeVirtEnabled := config.KubeVirt{
		Enabled: true,
		Sync: config.KubeVirtSync{
			DataVolumes:                      config.EnableSwitch{Enabled: true},
			VirtualMachineInstances:          config.EnableSwitch{Enabled: true},
			VirtualMachines:                  config.EnableSwitch{Enabled: true},
			VirtualMachineInstanceMigrations: config.EnableSwitch{Enabled: true},
		},
	}
	kubeVirtDisabled := config.KubeVirt{
		Enabled: false,
	}

	cases := []struct {
		name                      string
		customResourcesToHostSync map[string]config.SyncToHostCustomResource
		kubeVirtIntegration       config.KubeVirt
		checkErr                  func(t *testing.T, err error)
	}{
		{
			name: "valid config",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"some.other.resource": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtEnabled,
			checkErr:            noErrExpected,
		},
		{
			name: "datavolumes listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"datavolumes.cdi.kubevirt.io": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtEnabled,
			checkErr: expectErr("kube-virt integration is enabled but kube-virt custom resource " +
				"(datavolumes.cdi.kubevirt.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "virtualmachines listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"virtualmachines.kubevirt.io": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtEnabled,
			checkErr: expectErr("kube-virt integration is enabled but kube-virt custom resource " +
				"(virtualmachines.kubevirt.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "virtualmachineinstances listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"virtualmachineinstances.kubevirt.io": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtEnabled,
			checkErr: expectErr("kube-virt integration is enabled but kube-virt custom resource " +
				"(virtualmachineinstances.kubevirt.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "virtualmachineinstancemigrations listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"virtualmachineinstancemigrations.kubevirt.io": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtEnabled,
			checkErr: expectErr("kube-virt integration is enabled but kube-virt custom resource " +
				"(virtualmachineinstancemigrations.kubevirt.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "datavolumes listed in sync.toHost.customResources but kube-virt disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"datavolumes.cdi.kubevirt.io": {Enabled: true},
			},
			kubeVirtIntegration: kubeVirtDisabled,
			checkErr:            noErrExpected,
		},
		{
			name:                      "no custom resources and kube-virt enabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{},
			kubeVirtIntegration:       kubeVirtEnabled,
			checkErr:                  noErrExpected,
		},
		{
			name:                      "no custom resources and kube-virt disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{},
			kubeVirtIntegration:       kubeVirtDisabled,
			checkErr:                  noErrExpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateKubeVirtEnabled(tc.customResourcesToHostSync, tc.kubeVirtIntegration)
			tc.checkErr(t, err)
		})
	}
}

func TestValidateToHostSyncAndExternalSecretsIntegration(t *testing.T) {
	externalSecretsEnabled := config.ExternalSecrets{
		Enabled: true,
		Sync: config.ExternalSecretsSync{
			ToHost: config.ExternalSecretsSyncToHostConfig{
				Stores: config.EnableSwitchSelector{
					EnableSwitch: config.EnableSwitch{
						Enabled: true,
					},
				},
			},
			FromHost: config.ExternalSecretsSyncFromHostConfig{
				ClusterStores: config.EnableSwitchSelector{
					EnableSwitch: config.EnableSwitch{
						Enabled: true,
					},
				},
			},
		},
	}
	externalSecretsDisabled := config.ExternalSecrets{
		Enabled: false,
	}

	cases := []struct {
		name                       string
		customResourcesToHostSync  map[string]config.SyncToHostCustomResource
		externalSecretsIntegration config.ExternalSecrets
		checkErr                   func(t *testing.T, err error)
	}{
		{
			name: "valid config",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"some.other.resource": {Enabled: true},
			},
			externalSecretsIntegration: externalSecretsEnabled,
			checkErr:                   noErrExpected,
		},
		{
			name: "externalsecrets listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"externalsecrets.external-secrets.io": {Enabled: true},
			},
			externalSecretsIntegration: externalSecretsEnabled,
			checkErr: expectErr("external-secrets integration is enabled but external-secrets custom resource " +
				"(externalsecrets.external-secrets.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "secretstores listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"secretstores.external-secrets.io": {Enabled: true},
			},
			externalSecretsIntegration: externalSecretsEnabled,
			checkErr: expectErr("external-secrets integration is enabled but external-secrets custom resource " +
				"(secretstores.external-secrets.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "clustersecretstores listed in sync.toHost.customResources",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"clustersecretstores.external-secrets.io": {Enabled: true},
			},
			externalSecretsIntegration: externalSecretsEnabled,
			checkErr: expectErr("external-secrets integration is enabled but external-secrets custom resource " +
				"(clustersecretstores.external-secrets.io) is also set in the sync.toHost.customResources. " +
				"This is not supported, please remove the entry from sync.toHost.customResources"),
		},
		{
			name: "externalsecrets listed in sync.toHost.customResources but external-secrets disabled",
			customResourcesToHostSync: map[string]config.SyncToHostCustomResource{
				"externalsecrets.external-secrets.io": {Enabled: true},
			},
			externalSecretsIntegration: externalSecretsDisabled,
			checkErr:                   noErrExpected,
		},
		{
			name:                       "no custom resources and external-secrets enabled",
			customResourcesToHostSync:  map[string]config.SyncToHostCustomResource{},
			externalSecretsIntegration: externalSecretsEnabled,
			checkErr:                   noErrExpected,
		},
		{
			name:                       "no custom resources and external-secrets disabled",
			customResourcesToHostSync:  map[string]config.SyncToHostCustomResource{},
			externalSecretsIntegration: externalSecretsDisabled,
			checkErr:                   noErrExpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExternalSecretsEnabled(tc.customResourcesToHostSync, tc.externalSecretsIntegration)
			tc.checkErr(t, err)
		})
	}
}

func TestValidateToHostNamespaceSyncMappings(t *testing.T) {
	type testCase struct {
		name           string
		vclusterConfig *VirtualClusterConfig
		checkErr       func(t *testing.T, err error)
	}

	mockControlPlaneNamespaceForClientHelper := "vcluster-control-plane"
	t.Setenv("NAMESPACE", mockControlPlaneNamespaceForClientHelper)

	testCases := []testCase{
		{
			name: "Sync disabled",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled: false,
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Sync enabled, no mappings",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName are empty"),
		},
		{
			name: "Valid exact-to-exact mapping",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "hns1"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid exact-to-exact mapping with ${name} on host side",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "hns1-${name}"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid exact-to-exact mapping with ${name} on virtual side",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1-${name}": "hns1"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid exact-to-exact mapping with ${name} on both sides",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-${name}-foo": "hns-${name}-bar"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid pattern-to-pattern mapping",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "hns-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid pattern-to-pattern mapping with ${name} on host side prefix",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "hns-${name}-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid pattern-to-pattern mapping with ${name} on virtual side prefix",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-${name}-*": "hns-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid pattern-to-pattern mapping with ${name} on both prefixes",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-${name}-foo-*": "hns-${name}-bar-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Invalid: Host mapping is a catch-all wildcard",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"*": "*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: host pattern mappings must use a prefix before wildcard: *")},
		{
			name: "Invalid: Mismatched types (exact-to-pattern)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "hns-*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: 'vns1':'hns-*' has mismatched wildcard '*' usage - pattern must always map to another pattern"),
		},
		{
			name: "Invalid: Mismatched types (pattern-to-exact)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "hns1"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: 'vns-*':'hns1' has mismatched wildcard '*' usage - pattern must always map to another pattern"),
		},
		{
			name: "Invalid: Empty virtual exact name",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"": "hns1"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: virtual namespace cannot be empty"),
		},
		{
			name: "Invalid: Empty host exact name",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": ""}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: host namespace cannot be empty"),
		},
		{
			name: "Invalid: Virtual exact name not DNS compliant (uppercase)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"VNS1": "hns1"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: invalid virtual namespace name 'VNS1': a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		},
		{
			name: "Invalid: Host exact name not DNS compliant (too long)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": strings.Repeat("a", 64)}},
					}}},
				},
			},
			checkErr: expectErr(fmt.Sprintf("config.sync.toHost.namespaces.mappings.byName: invalid host namespace name '%s': must be no more than 63 characters", strings.Repeat("a", 64))),
		},
		{
			name: "Invalid: Virtual exact name with multiple ${name}",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-${name}-${name}": "hns1"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: virtual namespace 'vns-${name}-${name}' contains placeholder '${name}' multiple times"),
		},
		{
			name: "Invalid: Host exact name with unsupported placeholder",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "hns-${unsupported}"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: host namespace 'hns-${unsupported}' contains an unsupported placeholder; only '${name}' is allowed"),
		},
		{
			name: "Invalid: Virtual pattern with multiple '*'",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*-*": "hns-*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: virtual namespace pattern 'vns-*-*' must contain exactly one '*'"),
		},
		{
			name: "Invalid: Host pattern with '*' not at the end",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "hns-*-end"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: host namespace pattern 'hns-*-end' must have the wildcard '*' at the end"),
		},
		{
			name: "Invalid: Virtual pattern prefix too long",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{strings.Repeat("a", 33) + "*": "hns-*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: literal parts of virtual namespace pattern prefix '" + strings.Repeat("a", 33) + "' (from '" + strings.Repeat("a", 33) + "*') cannot be longer than 32 characters (literal length: 33)"),
		},
		{
			name: "Invalid: Host pattern prefix too long (with ${name})",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc", // length 7
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": strings.Repeat("a", 30) + "${name}" + strings.Repeat("b", 3) + "*"}}, // 30 + 7 + 3 = 40
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: literal parts of host namespace pattern prefix '" + strings.Repeat("a", 30) + "${name}" + strings.Repeat("b", 3) + "' (from '" + strings.Repeat("a", 30) + "${name}" + strings.Repeat("b", 3) + "*" + "') cannot be longer than 32 characters (literal length: 40)"),
		},
		{
			name: "Invalid: Virtual pattern prefix starts with '-'",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"-vns-*": "hns-*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: invalid virtual namespace pattern '-vns-*': a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		},
		{
			name: "Invalid: Host pattern prefix (with ${name}) ends with '-' after substitution",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "${name}-hns--*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Invalid: Virtual pattern prefix contains invalid char '.'",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns.foo-*": "hns-*"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: invalid virtual namespace pattern 'vns.foo-*': must not contain dots"),
		},
		{
			name: "Invalid: Virtual namespace mapping contains '/'",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns/foo": "hns"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: invalid virtual namespace name 'vns/foo': a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		},
		{
			name: "Invalid: Host namespace mapping contains '/'",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns": "hns/bar"}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: invalid host namespace name 'hns/bar': a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		},
		{
			name: "Invalid: Duplicate host namespace name (exact)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled: true,
						Mappings: config.FromHostMappings{ByName: map[string]string{
							"vns1": "common-hns",
							"vns2": "common-hns",
						}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: duplicate host namespace 'common-hns' found in mappings"),
		},
		{
			name: "Invalid: Duplicate host namespace name (pattern)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled: true,
						Mappings: config.FromHostMappings{ByName: map[string]string{
							"vns-a-*": "common-hns-*",
							"vns-b-*": "common-hns-*",
						}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: duplicate host namespace 'common-hns-*' found in mappings"),
		},
		{
			name: "Invalid: Duplicate host namespace name (with ${name} placeholder)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled: true,
						Mappings: config.FromHostMappings{ByName: map[string]string{
							"vns-x": "hns-${name}-duplicate",
							"vns-y": "hns-${name}-duplicate",
						}},
					}}},
				},
			},
			checkErr: expectErr("config.sync.toHost.namespaces.mappings.byName: duplicate host namespace 'hns-${name}-duplicate' found in mappings"),
		},
		{
			name: "Valid: No duplicate host namespaces (multiple different mappings)",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled: true,
						Mappings: config.FromHostMappings{ByName: map[string]string{
							"vns1":           "hns1",
							"vns2":           "hns2",
							"vns-pat-*":      "hns-pat-*",
							"vns-ph-${name}": "hns-ph-${name}",
						}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Invalid: Exact host namespace matches (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": mockControlPlaneNamespaceForClientHelper}},
					}}},
				},
			},
			checkErr: expectErr(fmt.Sprintf("config.sync.toHost.namespaces.mappings.byName: host namespace mapping '%s' conflicts with control plane namespace '%s'", mockControlPlaneNamespaceForClientHelper, mockControlPlaneNamespaceForClientHelper)),
		},
		{
			name: "Invalid: Exact host namespace with ${name} resolves to (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "control-plane",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "vcluster-${name}"}},
					}}},
				},
			},
			checkErr: expectErr(fmt.Sprintf("config.sync.toHost.namespaces.mappings.byName: host namespace mapping 'vcluster-${name}' conflicts with control plane namespace '%s'", mockControlPlaneNamespaceForClientHelper)),
		},
		{
			name: "Invalid: Pattern host namespace matches (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "vcluster-control*"}},
					}}},
				},
			},
			checkErr: expectErr(fmt.Sprintf("config.sync.toHost.namespaces.mappings.byName: host namespace pattern 'vcluster-control*' conflicts with control plane namespace '%s'", mockControlPlaneNamespaceForClientHelper)),
		},
		{
			name: "Invalid: Pattern host namespace with ${name} resolves to match (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "control",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "vcluster-${name}-plane*"}},
					}}},
				},
			},
			checkErr: expectErr(fmt.Sprintf("config.sync.toHost.namespaces.mappings.byName: host namespace pattern 'vcluster-${name}-plane*' conflicts with control plane namespace '%s'", mockControlPlaneNamespaceForClientHelper)),
		},
		{
			name: "Valid: No conflict if host namespace is different from (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns1": "different-host-ns"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid: No conflict if pattern host namespace does not match (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "different-host-prefix-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
		{
			name: "Valid: Pattern host namespace with ${name} does not match (mocked) clienthelper.CurrentNamespace()",
			vclusterConfig: &VirtualClusterConfig{
				Name: "test-vc",
				Config: config.Config{
					Sync: config.Sync{ToHost: config.SyncToHost{Namespaces: config.SyncToHostNamespaces{
						Enabled:  true,
						Mappings: config.FromHostMappings{ByName: map[string]string{"vns-*": "resolved-${name}-prefix-*"}},
					}}},
				},
			},
			checkErr: noErrExpected,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := namespaces.ValidateNamespaceSyncConfig(&tc.vclusterConfig.Config, tc.vclusterConfig.Name, mockControlPlaneNamespaceForClientHelper)
			tc.checkErr(t, err)
		})
	}
}

func expectErr(errMsg string) func(t *testing.T, err error) {
	return func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != errMsg {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func noErrExpected(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
