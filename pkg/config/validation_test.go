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
