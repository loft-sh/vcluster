package legacyconfig

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

type TestCaseMigration struct {
	Name string

	Distro string
	In     string

	Expected    string
	ExpectedErr string
}

func TestMigration(t *testing.T) {
	testCases := []TestCaseMigration{
		{
			Name:   "Simple k3s",
			Distro: "k3s",
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "k3s with deprecated serviceCIDR",
			Distro: "k3s",
			In: `
serviceCIDR: 10.96.0.0/16
`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "Simple k0s",
			Distro: "k0s",
			Expected: `controlPlane:
  distro:
    k0s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "Plugin k3s",
			Distro: "k3s",
			In: `plugin:
  test:
    version: v2`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
plugin:
  test:
    version: v2`,
		},
		{
			Name:   "Simple k8s",
			Distro: "k8s",
			In: `sync:
  ingresses:
    enabled: true`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
  distro:
    k8s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
sync:
  fromHost:
    ingressClasses:
      enabled: true
  toHost:
    ingresses:
      enabled: true`,
		},
		{
			Name:   "Simple eks",
			Distro: "eks",
			In: `sync:
  ingresses:
    enabled: true`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
  distro:
    eks:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
sync:
  fromHost:
    ingressClasses:
      enabled: true
  toHost:
    ingresses:
      enabled: true`,
		},
		{
			Name:   "generic sync example",
			Distro: "k3s",
			In: `multiNamespaceMode:
  enabled: true
sync:
  generic:
    role:
      extraRules:
        - apiGroups: ["cert-manager.io"]
          resources: ["issuers", "certificates"]
          verbs: ["create", "delete", "patch", "update", "get", "list", "watch"]
    clusterRole:
      extraRules:
        - apiGroups: ["apiextensions.k8s.io"]
          resources: ["customresourcedefinitions"]
          verbs: ["get", "list", "watch"]
    config: |-
      version: v1beta1
      export:
        - apiVersion: cert-manager.io/v1
          kind: Issuer
        - apiVersion: cert-manager.io/v1
          kind: Certificate
      import:
        - kind: Secret
          apiVersion: v1`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
experimental:
  genericSync:
    export:
    - apiVersion: cert-manager.io/v1
      kind: Issuer
    - apiVersion: cert-manager.io/v1
      kind: Certificate
    import:
    - apiVersion: v1
      kind: Secret
    version: v1beta1
  multiNamespaceMode:
    enabled: true`,
		},
		{
			Name:   "persistence false",
			Distro: "k3s",
			In: `syncer:
  storage:
    persistence: false`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    persistence:
      volumeClaim:
        enabled: false
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "vcluster env",
			Distro: "k3s",
			In: `vcluster:
  env:
  - name: K3S_DATASTORE_ENDPOINT
    value: postgres://username:password@hostname:5432/k3s`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
      env:
      - name: K3S_DATASTORE_ENDPOINT
        value: postgres://username:password@hostname:5432/k3s
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "high availability",
			Distro: "k8s",
			In: `syncer:
  replicas: 3
etcd:
  replicas: 3
coredns:
  replicas: 3`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
        statefulSet:
          highAvailability:
            replicas: 3
  coredns:
    deployment:
      replicas: 3
  distro:
    k8s:
      enabled: true
  statefulSet:
    highAvailability:
      replicas: 3
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "fallback host dns",
			Distro: "k0s",
			In: `fallbackHostDns: true
pro: true`,
			Expected: `controlPlane:
  distro:
    k0s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
networking:
  advanced:
    fallbackHostCluster: true
pro: true`,
		},
		{
			Name:   "isolated mode",
			Distro: "k0s",
			In: `isolation:
  enabled: true
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true
  limitRange:
    enabled: true
  networkPolicy:
    enabled: false`,
			Expected: `controlPlane:
  distro:
    k0s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
policies:
  limitRange:
    enabled: true
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true`,
		},
		{
			Name:   "convert flags",
			Distro: "k0s",
			In: `syncer:
  extraArgs:
  - --tls-san=my-vcluster.example.com
  - --service-account-token-secrets=true
  - --mount-physical-host-paths=true
  - --sync-all-nodes`,
			Expected: `controlPlane:
  distro:
    k0s:
      enabled: true
  hostPathMapper:
    enabled: true
  proxy:
    extraSANs:
    - my-vcluster.example.com
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
sync:
  fromHost:
    nodes:
      selector:
        all: true
  toHost:
    pods:
      useSecretsForSATokens: true`,
		},
		{
			Name:   "convert deprecated host-path-mapper flag",
			Distro: "k0s",
			In: `syncer:
  extraArgs:
  - --rewrite-host-paths=true
`,
			Expected: `controlPlane:
  distro:
    k0s:
      enabled: true
  hostPathMapper:
    enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
`,
		},
		{
			Name:   "embedded etcd",
			Distro: "k8s",
			In: `embeddedEtcd:
  enabled: true`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      embedded:
        enabled: true
  distro:
    k8s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "scheduler",
			Distro: "k3s",
			In: `sync:
  csistoragecapacities:
    enabled: false
  csinodes:
    enabled: false
  nodes:
    enableScheduler: true`,
			Expected: `controlPlane:
  advanced:
    virtualScheduler:
      enabled: true
  distro:
    k3s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady
sync:
  fromHost:
    csiNodes:
      enabled: false
    csiStorageCapacities:
      enabled: false`,
		},
		{
			Name:   "image",
			Distro: "k3s",
			In: `vcluster:
  image: my-registry.com:5000/private/private:v0.0.1
syncer:
  image: loft-sh/test:abc`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
      image:
        registry: my-registry.com:5000
        repository: private/private
        tag: v0.0.1
  statefulSet:
    image:
      registry: ""
      repository: loft-sh/test
      tag: abc
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "binariesVolume",
			Distro: "k3s",
			In: `syncer: 
  storage: 
    binariesVolume: 
    - name: binaries
      persistentVolumeClaim:
        claimName: my-pvc`,
			Expected: `controlPlane:
  distro:
    k3s:
      enabled: true
  statefulSet:
    persistence:
      binariesVolume:
      - name: binaries
        persistentVolumeClaim:
          claimName: my-pvc
    scheduling:
      podManagementPolicy: OrderedReady`,
			ExpectedErr: "",
		},
	}

	for _, testCase := range testCases {
		out, err := MigrateLegacyConfig(testCase.Distro, testCase.In)
		if err != nil {
			if testCase.ExpectedErr != "" && testCase.ExpectedErr == err.Error() {
				continue
			}

			t.Fatalf("Test case %s failed with: %v", testCase.Name, err)
		}

		if strings.TrimSpace(testCase.Expected) != strings.TrimSpace(out) {
			t.Log(out)
		}
		assert.Equal(t, strings.TrimSpace(testCase.Expected), strings.TrimSpace(out), testCase.Name)
	}
}
