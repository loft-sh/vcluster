package coredns

import (
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/config"
	"gotest.tools/v3/assert"
)

func mustNewDefaultConfig(changes ...func(*config.Config)) *config.Config {
	config, err := config.NewDefaultConfig()
	if err != nil {
		panic(err)
	}

	for _, change := range changes {
		change(config)
	}
	return config
}

func TestProcessManifests(t *testing.T) {
	tests := []struct {
		name   string
		vars   map[string]interface{}
		config *config.Config
		want   string
	}{
		{
			name:   "default",
			config: mustNewDefaultConfig(),
			vars: map[string]interface{}{
				"IMAGE":        "my-image",
				"RUN_AS_USER":  1001,
				"RUN_AS_GROUP": 1001,
			},
			want: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
      - services
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
      - discovery.k8s.io
    resources:
      - endpointslices
    verbs:
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
  - kind: ServiceAccount
    name: coredns
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |-
    .:1053 {
        errors
        health
        ready
        rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        hosts /etc/coredns/NodeHosts {
            ttl 60
            reload 15s
            fallthrough
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }

    import /etc/coredns/custom/*.server
  NodeHosts: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: vcluster-kube-dns
  template:
    metadata:
      labels:
        k8s-app: vcluster-kube-dns
    spec:
      priorityClassName: ""
      serviceAccountName: coredns
      nodeSelector:
        kubernetes.io/os: linux
      topologySpreadConstraints:
        - labelSelector:
            matchLabels:
              k8s-app: vcluster-kube-dns
          maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
      containers:
        - name: coredns
          image: my-image
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: 1000m
              memory: 170Mi
            requests:
              cpu: 20m
              memory: 64Mi
          args: [ "-conf", "/etc/coredns/Corefile" ]
          volumeMounts:
            - name: config-volume
              mountPath: /etc/coredns
              readOnly: true
            - name: custom-config-volume
              mountPath: /etc/coredns/custom
              readOnly: true
          securityContext:
            runAsNonRoot: true
            runAsUser: 1001
            runAsGroup: 1001
            allowPrivilegeEscalation: false
            capabilities:
              add:
                - NET_BIND_SERVICE
              drop:
                - ALL
            readOnlyRootFilesystem: true
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 60
            periodSeconds: 10
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: 8181
              scheme: HTTP
            initialDelaySeconds: 0
            periodSeconds: 2
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
              - key: Corefile
                path: Corefile
              - key: NodeHosts
                path: NodeHosts
        - name: custom-config-volume
          configMap:
            name: coredns-custom
            optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  type: ClusterIP
  selector:
    k8s-app: vcluster-kube-dns
  ports:
    - name: dns
      port: 53
      targetPort: 1053
      protocol: UDP
    - name: dns-tcp
      port: 53
      targetPort: 1053
      protocol: TCP
    - name: metrics
      port: 9153
      protocol: TCP`,
		},
		{
			name: "overwrite",
			vars: map[string]interface{}{},
			config: &config.Config{
				ControlPlane: config.ControlPlane{
					CoreDNS: config.CoreDNS{
						Enabled:            true,
						OverwriteManifests: "abc",
					},
				},
			},
			want: `abc`,
		},
		{
			name: "embedded",
			vars: map[string]interface{}{},
			config: mustNewDefaultConfig(func(config *config.Config) {
				config.ControlPlane.CoreDNS.Embedded = true
			}),
			want: `apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  NodeHosts: ""
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  type: ClusterIP
  ports:
    - name: dns
      port: 53
      targetPort: 1053
      protocol: UDP
    - name: dns-tcp
      port: 53
      targetPort: 1053
      protocol: TCP
    - name: metrics
      port: 9153
      protocol: TCP`,
		},
		{
			name: "should correctly set default coredns clusterdomain",
			vars: map[string]interface{}{
				"IMAGE":        "my-image",
				"RUN_AS_USER":  1001,
				"RUN_AS_GROUP": 1001,
			},
			config: mustNewDefaultConfig(func(config *config.Config) {
				config.Networking.Advanced.ClusterDomain = "cluster.local"
			}),
			want: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
      - services
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
      - discovery.k8s.io
    resources:
      - endpointslices
    verbs:
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
  - kind: ServiceAccount
    name: coredns
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |-
    .:1053 {
        errors
        health
        ready
        rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        hosts /etc/coredns/NodeHosts {
            ttl 60
            reload 15s
            fallthrough
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
		reload
        loadbalance
    }

    import /etc/coredns/custom/*.server
  NodeHosts: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: vcluster-kube-dns
  template:
    metadata:
      labels:
        k8s-app: vcluster-kube-dns
    spec:
      priorityClassName: ""
      serviceAccountName: coredns
      nodeSelector:
        kubernetes.io/os: linux
      topologySpreadConstraints:
        - labelSelector:
            matchLabels:
              k8s-app: vcluster-kube-dns
          maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
      containers:
        - name: coredns
          image: my-image
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: 1000m
              memory: 170Mi
            requests:
              cpu: 20m
              memory: 64Mi
          args: [ "-conf", "/etc/coredns/Corefile" ]
          volumeMounts:
            - name: config-volume
              mountPath: /etc/coredns
              readOnly: true
            - name: custom-config-volume
              mountPath: /etc/coredns/custom
              readOnly: true
          securityContext:
            runAsNonRoot: true
            runAsUser: 1001
            runAsGroup: 1001
            allowPrivilegeEscalation: false
            capabilities:
              add:
                - NET_BIND_SERVICE
              drop:
                - ALL
            readOnlyRootFilesystem: true
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 60
            periodSeconds: 10
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: 8181
              scheme: HTTP
            initialDelaySeconds: 0
            periodSeconds: 2
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
              - key: Corefile
                path: Corefile
              - key: NodeHosts
                path: NodeHosts
        - name: custom-config-volume
          configMap:
            name: coredns-custom
            optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  type: ClusterIP
  selector:
    k8s-app: vcluster-kube-dns
  ports:
    - name: dns
      port: 53
      targetPort: 1053
      protocol: UDP
    - name: dns-tcp
      port: 53
      targetPort: 1053
      protocol: TCP
    - name: metrics
      port: 9153
      protocol: TCP`,
		},
		{
			name: "should correctly set custom coredns clusterdomain",
			vars: map[string]interface{}{
				"IMAGE":        "my-image",
				"RUN_AS_USER":  1001,
				"RUN_AS_GROUP": 1001,
			},
			config: mustNewDefaultConfig(func(config *config.Config) {
				config.Networking.Advanced.ClusterDomain = "custom.local"
			}),
			want: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
      - services
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
      - discovery.k8s.io
    resources:
      - endpointslices
    verbs:
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
  - kind: ServiceAccount
    name: coredns
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |-
    .:1053 {
        errors
        health
        ready
        rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
        kubernetes custom.local cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        hosts /etc/coredns/NodeHosts {
            ttl 60
            reload 15s
            fallthrough
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
		reload
        loadbalance
    }

    import /etc/coredns/custom/*.server
  NodeHosts: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: vcluster-kube-dns
  template:
    metadata:
      labels:
        k8s-app: vcluster-kube-dns
    spec:
      priorityClassName: ""
      serviceAccountName: coredns
      nodeSelector:
        kubernetes.io/os: linux
      topologySpreadConstraints:
        - labelSelector:
            matchLabels:
              k8s-app: vcluster-kube-dns
          maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
      containers:
        - name: coredns
          image: my-image
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: 1000m
              memory: 170Mi
            requests:
              cpu: 20m
              memory: 64Mi
          args: [ "-conf", "/etc/coredns/Corefile" ]
          volumeMounts:
            - name: config-volume
              mountPath: /etc/coredns
              readOnly: true
            - name: custom-config-volume
              mountPath: /etc/coredns/custom
              readOnly: true
          securityContext:
            runAsNonRoot: true
            runAsUser: 1001
            runAsGroup: 1001
            allowPrivilegeEscalation: false
            capabilities:
              add:
                - NET_BIND_SERVICE
              drop:
                - ALL
            readOnlyRootFilesystem: true
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 60
            periodSeconds: 10
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: 8181
              scheme: HTTP
            initialDelaySeconds: 0
            periodSeconds: 2
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
              - key: Corefile
                path: Corefile
              - key: NodeHosts
                path: NodeHosts
        - name: custom-config-volume
          configMap:
            name: coredns-custom
            optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  type: ClusterIP
  selector:
    k8s-app: vcluster-kube-dns
  ports:
    - name: dns
      port: 53
      targetPort: 1053
      protocol: UDP
    - name: dns-tcp
      port: 53
      targetPort: 1053
      protocol: TCP
    - name: metrics
      port: 9153
      protocol: TCP`,
		},
		{
			name: "should correctly set coreDNS security context",
			vars: map[string]interface{}{
				"IMAGE": "my-image",
			},
			config: mustNewDefaultConfig(func(config *config.Config) {
				config.ControlPlane.CoreDNS.Security.PodSecurityContext = map[string]interface{}{
					"runAsUser":  1042,
					"runAsGroup": 2042,
					"fsGroup":    3042,
				}
				config.ControlPlane.CoreDNS.Security.ContainerSecurityContext = map[string]interface{}{
					"runAsUser":                1142,
					"runAsGroup":               2242,
					"allowPrivilegeEscalation": false,
					"capabilities": map[string]interface{}{
						"drop": []string{"ALL"},
						"add":  []string{"NET_BIND_SERVICE"},
					},
				}
			}),
			want: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
rules:
  - apiGroups:
      - ""
    resources:
      - endpoints
      - services
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
      - discovery.k8s.io
    resources:
      - endpointslices
    verbs:
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
  - kind: ServiceAccount
    name: coredns
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |-
    .:1053 {
        errors
        health
        ready
        rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        hosts /etc/coredns/NodeHosts {
            ttl 60
            reload 15s
            fallthrough
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
		reload
        loadbalance
    }

    import /etc/coredns/custom/*.server
  NodeHosts: ""
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: vcluster-kube-dns
  template:
    metadata:
      labels:
        k8s-app: vcluster-kube-dns
    spec:
      priorityClassName: ""
      serviceAccountName: coredns
      nodeSelector:
        kubernetes.io/os: linux
      topologySpreadConstraints:
        - labelSelector:
            matchLabels:
              k8s-app: vcluster-kube-dns
          maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
      securityContext:
        fsGroup: 3042
        runAsGroup: 2042
        runAsUser: 1042
      containers:
        - name: coredns
          image: my-image
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: 1000m
              memory: 170Mi
            requests:
              cpu: 20m
              memory: 64Mi
          args: [ "-conf", "/etc/coredns/Corefile" ]
          volumeMounts:
            - name: config-volume
              mountPath: /etc/coredns
              readOnly: true
            - name: custom-config-volume
              mountPath: /etc/coredns/custom
              readOnly: true
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              add:
              - NET_BIND_SERVICE
              drop:
              - ALL
            runAsGroup: 2242
            runAsUser: 1142
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 60
            periodSeconds: 10
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: 8181
              scheme: HTTP
            initialDelaySeconds: 0
            periodSeconds: 2
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
              - key: Corefile
                path: Corefile
              - key: NodeHosts
                path: NodeHosts
        - name: custom-config-volume
          configMap:
            name: coredns-custom
            optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: vcluster-kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  type: ClusterIP
  selector:
    k8s-app: vcluster-kube-dns
  ports:
    - name: dns
      port: 53
      targetPort: 1053
      protocol: UDP
    - name: dns-tcp
      port: 53
      targetPort: 1053
      protocol: TCP
    - name: metrics
      port: 9153
      protocol: TCP`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifests, err := processManifests(test.vars, test.config)
			if err != nil {
				t.Fatalf("ProcessManifests failed: %v", err)
			}
			assert.Equal(t, strings.TrimSpace(test.want), strings.TrimSpace(string(manifests)))
		})
	}
}

func TestProcessCorefile(t *testing.T) {
	tests := []struct {
		name   string
		vars   map[string]interface{}
		config *config.Config
		want   string
	}{
		{
			name: "embedded",
			vars: map[string]interface{}{},
			config: &config.Config{
				ControlPlane: config.ControlPlane{
					CoreDNS: config.CoreDNS{
						Enabled:  true,
						Embedded: true,
					},
				},
			},
			want: `.:1053 {
    errors
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        kubeconfig /data/vcluster/admin.conf
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }
    hosts /etc/coredns/NodeHosts {
        ttl 60
        reload 15s
        fallthrough
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
	reload
    loadbalance
}

import /etc/coredns/custom/*.server`,
		},
		{
			name: "overwrite",
			vars: map[string]interface{}{},
			config: &config.Config{
				ControlPlane: config.ControlPlane{
					CoreDNS: config.CoreDNS{
						Enabled:         true,
						Embedded:        true,
						OverwriteConfig: "abc",
					},
				},
			},
			want: `abc`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			corefile, err := ProcessCorefile(test.vars, test.config)
			if err != nil {
				t.Fatalf("ProcessCorefile failed: %v", err)
			}
			assert.Equal(t, strings.TrimSpace(test.want), strings.TrimSpace(corefile))
		})
	}
}
