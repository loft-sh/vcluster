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
			Distro: "k8s",
			In: `sync:
  csistoragecapacities:
    enabled: false
  csinodes:
    enabled: false
  nodes:
    enableScheduler: true`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
  distro:
    k8s:
      enabled: true
      scheduler:
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
			Name:   "scheduler extra args",
			Distro: "k8s",
			In: `syncer:
  extraArgs:
  - --enable-scheduler`,
			Expected: `controlPlane:
  advanced:
    virtualScheduler:
      enabled: true
  backingStore:
    etcd:
      deploy:
        enabled: true
  distro:
    k8s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
		},
		{
			Name:   "quotas",
			Distro: "k8s",
			In: `isolation:
  enabled: true
  resourceQuota:
    enabled: true
    quota:
      limits.cpu: 16`,
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
policies:
  limitRange:
    enabled: true
  networkPolicy:
    enabled: true
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true
    quota:
      limits.cpu: 16`,
			ExpectedErr: "",
		},
		{
			Name:   "resources",
			Distro: "k8s",
			In: `syncer:
  resources:
    limits:
      memory: 10Gi`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      deploy:
        enabled: true
  distro:
    k8s:
      enabled: true
  statefulSet:
    resources:
      limits:
        memory: 10Gi
    scheduling:
      podManagementPolicy: OrderedReady`,
			ExpectedErr: "",
		},
		{
			Name:   "k8s already migrated to correct format",
			Distro: "k8s",
			In: `sync:
  fromHost:
    nodes:
      enabled: false
  toHost:
    serviceAccounts:
      enabled: false
controlPlane:
  distro:
    k8s:
      enabled: true
  statefulSet:
    scheduling:
      podManagementPolicy: OrderedReady`,
			ExpectedErr: "migrate legacy k8s values: config is already in correct format",
		},
		{
			Name:   "statefulset affinity added",
			Distro: "k8s",
			In: `isolation:
  # nodeProxyPermission:
  #  enabled: true
  enabled: true
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true
    quota:
      count/endpoints: null
      count/pods: null
      count/services: null
      count/configmaps: null
      count/secrets: null
      count/persistentvolumeclaims: null
      limits.cpu: 256
      limits.memory: 1Ti
      requests.storage: 10Ti
      requests.ephemeral-storage: null
      requests.memory: 128Gi
      requests.cpu: 120
      services.loadbalancers: null
      services.nodeports: null
  limitRange:
    enabled: true
    defaultRequest:
      cpu: 24m
      memory: 32Mi
      ephemeral-storage: null
    default:
      ephemeral-storage: null
      memory: 2Gi
      cpu: 512m
    # max:
    #  cpu: 32
    #  memory: 64Gi
    #  ephemeral-storage: 512Gi
  networkPolicy:
    enabled: false
storage:
  className: px-pool
sync:
  secrets:
    enabled: true
  nodes:
    enabled: true
  networkpolicies:
    enabled: true
  hoststorageclasses:
    enabled: true
# enableHA: true
embeddedEtcd:
  enabled: true
syncer:
  resources:
    limits:
      cpu: '8'
      ephemeral-storage: 10Gi
      memory: 10Gi
  # extraArgs:
  #  - '--sync-labels=namespace,aussiebb.io/,..aussiebb.io/'
  replicas: 3
  labels:
    aussiebb.io/profile: "true"
  storage:
    size: 50Gi
    className: px-pool-etcd
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app
                operator: In
                values:
                  - vcluster
          topologyKey: "kubernetes.io/hostname"
coredns:
  replicas: 3
  resources:
    limits:
      cpu: '2'
      memory: '1Gi'
api:
  extraArgs:
    - "-v=4"`,
			Expected: `controlPlane:
  backingStore:
    etcd:
      embedded:
        enabled: true
  coredns:
    deployment:
      replicas: 3
      resources:
        limits:
          cpu: "2"
          memory: 1Gi
  distro:
    k8s:
      apiServer:
        extraArgs:
        - -v=4
      enabled: true
  statefulSet:
    highAvailability:
      replicas: 3
    persistence:
      volumeClaim:
        size: 50Gi
        storageClass: px-pool-etcd
    resources:
      limits:
        cpu: "8"
        memory: 10Gi
    scheduling:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - vcluster
            topologyKey: kubernetes.io/hostname
      podManagementPolicy: OrderedReady
policies:
  limitRange:
    default:
      cpu: 512m
      ephemeral-storage: null
      memory: 2Gi
    defaultRequest:
      cpu: 24m
      ephemeral-storage: null
      memory: 32Mi
    enabled: true
  podSecurityStandard: baseline
  resourceQuota:
    enabled: true
    quota:
      count/configmaps: null
      count/endpoints: null
      count/persistentvolumeclaims: null
      count/pods: null
      count/secrets: null
      count/services: null
      limits.cpu: 256
      limits.memory: 1Ti
      requests.cpu: 120
      requests.ephemeral-storage: null
      requests.memory: 128Gi
      requests.storage: 10Ti
      services.loadbalancers: null
      services.nodeports: null
sync:
  fromHost:
    nodes:
      enabled: true
    storageClasses:
      enabled: true
  toHost:
    networkPolicies:
      enabled: true`,
			ExpectedErr: "",
		},
		{
			Name:   "export kubeconfig with custom context and server",
			Distro: "k8s",
			In: `syncer:
  extraArgs:
  - --kube-config-context-name=my-context
  - --out-kube-config-server=https://my-vcluster.example.com`,
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
exportKubeConfig:
  context: my-context
  server: https://my-vcluster.example.com`,
		},
		{
			Name:   "export additional kubeconfig secret",
			Distro: "k8s",
			In: `syncer:
  extraArgs:
  - --out-kube-config-secret=my-secret
  - --out-kube-config-secret-namespace=my-namespace`,
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
exportKubeConfig:
  additionalSecrets:
  - name: my-secret
    namespace: my-namespace`,
		},
		{
			Name:   "export additional kubeconfig secret with custom context and server",
			Distro: "k8s",
			In: `syncer:
  extraArgs:
  - --kube-config-context-name=my-context
  - --out-kube-config-server=https://my-vcluster.example.com
  - --out-kube-config-secret=my-secret
  - --out-kube-config-secret-namespace=my-namespace`,
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
exportKubeConfig:
  additionalSecrets:
  - context: my-context
    name: my-secret
    namespace: my-namespace
    server: https://my-vcluster.example.com
  context: my-context
  server: https://my-vcluster.example.com`,
		},
		{
			Name:   "migrate rewriteHosts image",
			Distro: "k8s",
			In: `syncer:
  extraArgs:
  - --override-hosts-container-image=my.registry:5000/some/repo:a-tag`,
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
  toHost:
    pods:
      rewriteHosts:
        initContainer:
          image:
            registry: my.registry:5000
            repository: some/repo
            tag: a-tag
`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			out, err := MigrateLegacyConfig(testCase.Distro, testCase.In)
			if err != nil {
				if testCase.ExpectedErr != "" && testCase.ExpectedErr == err.Error() {
					return
				}

				t.Fatalf("Test case %s failed with: %v", testCase.Name, err)
			}

			if strings.TrimSpace(testCase.Expected) != strings.TrimSpace(out) {
				t.Log(out)
			}
			assert.Equal(t, strings.TrimSpace(testCase.Expected), strings.TrimSpace(out), testCase.Name)
		})
	}
}
