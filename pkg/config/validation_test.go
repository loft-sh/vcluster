package config

import (
	"testing"

	"github.com/loft-sh/vcluster/config"
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
	noErrExpected := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	expectErr := func(errMsg string) func(t *testing.T, err error) {
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
		expectedError    string
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
			expectedError: exportKubeConfigBothSecretAndAdditionalSecretsSetError,
		},
		{
			name: "Setting empty additional secret is not valid",
			exportKubeConfig: config.ExportKubeConfig{
				AdditionalSecrets: []config.ExportKubeConfigAdditionalSecretReference{
					{},
				},
			},
			expectedError: exportKubeConfigAdditionalSecretWithoutNameAndNamespace,
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
			expectedError: exportKubeConfigAdditionalSecretWithoutNameAndNamespace,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actualError := validateExportKubeConfig(tc.exportKubeConfig)
			if tc.expectedError == "" && actualError != nil {
				t.Errorf("expected validation to pass, but got error: %v", actualError)
			}
			if tc.expectedError != "" {
				if actualError == nil {
					t.Errorf("expected validation to fail with error %q, but it passed", tc.expectedError)
				} else if actualError.Error() != tc.expectedError {
					t.Errorf("expected to get error %q, but instead got other error: %v", tc.expectedError, actualError)
				}
			}
		})
	}
}
