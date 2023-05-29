rbac:
  create: true

podSecurityPolicy:
  enabled: false

imagePullSecrets: []
# - name: "image-pull-secret"

## Define serviceAccount names for components. Defaults to component's fully qualified name.
##
serviceAccounts:
  server:
    create: true
    name: ""
    annotations: {}

## Monitors ConfigMap changes and POSTs to a URL
## Ref: https://github.com/prometheus-operator/prometheus-operator/tree/main/cmd/prometheus-config-reloader
##
configmapReload:
  ## URL for configmap-reload to use for reloads
  ##
  reloadUrl: ""

  ## env sets environment variables to pass to the container. Can be set as name/value pairs,
  ## read from secrets or configmaps.
  env: []
    # - name: SOMEVAR
    #   value: somevalue
    # - name: PASSWORD
    #   valueFrom:
    #     secretKeyRef:
    #       name: mysecret
    #       key: password
    #       optional: false

  prometheus:
    ## If false, the configmap-reload container will not be deployed
    ##
    enabled: true

    ## configmap-reload container name
    ##
    name: configmap-reload

    ## configmap-reload container image
    ##
    image:
      repository: quay.io/prometheus-operator/prometheus-config-reloader
      tag: v0.65.1
      # When digest is set to a non-empty value, images will be pulled by digest (regardless of tag value).
      digest: ""
      pullPolicy: IfNotPresent

    # containerPort: 9533

    ## Additional configmap-reload container arguments
    ##
    extraArgs: {}

    ## Additional configmap-reload volume directories
    ##
    extraVolumeDirs: []

    ## Additional configmap-reload mounts
    ##
    extraConfigmapMounts: []
      # - name: prometheus-alerts
      #   mountPath: /etc/alerts.d
      #   subPath: ""
      #   configMap: prometheus-alerts
      #   readOnly: true

    ## Security context to be added to configmap-reload container
    containerSecurityContext: {}

    ## configmap-reload resource requests and limits
    ## Ref: http://kubernetes.io/docs/user-guide/compute-resources/
    ##
    resources: {}

server:
  ## Prometheus server container name
  ##
  name: server

  ## Use a ClusterRole (and ClusterRoleBinding)
  ## - If set to false - we define a RoleBinding in the defined namespaces ONLY
  ##
  ## NB: because we need a Role with nonResourceURL's ("/metrics") - you must get someone with Cluster-admin privileges to define this role for you, before running with this setting enabled.
  ##     This makes prometheus work - for users who do not have ClusterAdmin privs, but wants prometheus to operate on their own namespaces, instead of clusterwide.
  ##
  ## You MUST also set namespaces to the ones you have access to and want monitored by Prometheus.
  ##
  # useExistingClusterRoleName: nameofclusterrole

  ## If set it will override prometheus.server.fullname value for ClusterRole and ClusterRoleBinding
  ##
  clusterRoleNameOverride: ""

  ## namespaces to monitor (instead of monitoring all - clusterwide). Needed if you want to run without Cluster-admin privileges.
  # namespaces:
  #   - yournamespace

  # sidecarContainers - add more containers to prometheus server
  # Key/Value where Key is the sidecar `- name: <Key>`
  # Example:
  #   sidecarContainers:
  #      webserver:
  #        image: nginx
  sidecarContainers: {}

  # sidecarTemplateValues - context to be used in template for sidecarContainers
  # Example:
  #   sidecarTemplateValues: *your-custom-globals
  #   sidecarContainers:
  #     webserver: |-
  #       {{ include "webserver-container-template" . }}
  # Template for `webserver-container-template` might looks like this:
  #   image: "{{ .Values.server.sidecarTemplateValues.repository }}:{{ .Values.server.sidecarTemplateValues.tag }}"
  #   ...
  #
  sidecarTemplateValues: {}

  ## Prometheus server container image
  ##
  image:
    repository: quay.io/prometheus/prometheus
    # if not set appVersion field from Chart.yaml is used
    tag: ""
    # When digest is set to a non-empty value, images will be pulled by digest (regardless of tag value).
    digest: ""
    pullPolicy: IfNotPresent

  ## Prometheus server command
  ##
  command: []

  ## prometheus server priorityClassName
  ##
  priorityClassName: ""

  ## EnableServiceLinks indicates whether information about services should be injected
  ## into pod's environment variables, matching the syntax of Docker links.
  ## WARNING: the field is unsupported and will be skipped in K8s prior to v1.13.0.
  ##
  enableServiceLinks: true

  ## The URL prefix at which the container can be accessed. Useful in the case the '-web.external-url' includes a slug
  ## so that the various internal URLs are still able to access as they are in the default case.
  ## (Optional)
  prefixURL: ""

  ## External URL which can access prometheus
  ## Maybe same with Ingress host name
  baseURL: ""

  ## Additional server container environment variables
  ##
  ## You specify this manually like you would a raw deployment manifest.
  ## This means you can bind in environment variables from secrets.
  ##
  ## e.g. static environment variable:
  ##  - name: DEMO_GREETING
  ##    value: "Hello from the environment"
  ##
  ## e.g. secret environment variable:
  ## - name: USERNAME
  ##   valueFrom:
  ##     secretKeyRef:
  ##       name: mysecret
  ##       key: username
  env: []

  # List of flags to override default parameters, e.g:
  # - --enable-feature=agent
  # - --storage.agent.retention.max-time=30m
  defaultFlagsOverride: []

  extraFlags:
    - web.enable-lifecycle
    ## web.enable-admin-api flag controls access to the administrative HTTP API which includes functionality such as
    ## deleting time series. This is disabled by default.
    # - web.enable-admin-api
    ##
    ## storage.tsdb.no-lockfile flag controls BD locking
    # - storage.tsdb.no-lockfile
    ##
    ## storage.tsdb.wal-compression flag enables compression of the write-ahead log (WAL)
    # - storage.tsdb.wal-compression

  ## Path to a configuration file on prometheus server container FS
  configPath: /etc/config/prometheus.yml

  ### The data directory used by prometheus to set --storage.tsdb.path
  ### When empty server.persistentVolume.mountPath is used instead
  storagePath: ""

  global:
    ## How frequently to scrape targets by default
    ##
    scrape_interval: 1m
    ## How long until a scrape request times out
    ##
    scrape_timeout: 10s
    ## How frequently to evaluate rules
    ##
    evaluation_interval: 1m
  ## https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write
  ##
  remoteWrite: []
  ## https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_read
  ##
  remoteRead: []

  ## https://prometheus.io/docs/prometheus/latest/configuration/configuration/#tsdb
  ##
  tsdb: {}
    # out_of_order_time_window: 0s

  ## https://prometheus.io/docs/prometheus/latest/configuration/configuration/#exemplars
  ## Must be enabled via --enable-feature=exemplar-storage
  ##
  exemplars: {}
    # max_exemplars: 100000

  ## Custom HTTP headers for Liveness/Readiness/Startup Probe
  ##
  ## Useful for providing HTTP Basic Auth to healthchecks
  probeHeaders: []
    # - name: "Authorization"
    #   value: "Bearer ABCDEabcde12345"

  ## Additional Prometheus server container arguments
  ##
  extraArgs: {}

  ## Additional InitContainers to initialize the pod
  ##
  extraInitContainers: []

  ## Additional Prometheus server Volume mounts
  ##
  extraVolumeMounts: []

  ## Additional Prometheus server Volumes
  ##
  extraVolumes: []

  ## Additional Prometheus server hostPath mounts
  ##
  extraHostPathMounts: []
    # - name: certs-dir
    #   mountPath: /etc/kubernetes/certs
    #   subPath: ""
    #   hostPath: /etc/kubernetes/certs
    #   readOnly: true

  extraConfigmapMounts: []
    # - name: certs-configmap
    #   mountPath: /prometheus
    #   subPath: ""
    #   configMap: certs-configmap
    #   readOnly: true

  ## Additional Prometheus server Secret mounts
  # Defines additional mounts with secrets. Secrets must be manually created in the namespace.
  extraSecretMounts: []
    # - name: secret-files
    #   mountPath: /etc/secrets
    #   subPath: ""
    #   secretName: prom-secret-files
    #   readOnly: true

  ## ConfigMap override where fullname is {{.Release.Name}}-{{.Values.server.configMapOverrideName}}
  ## Defining configMapOverrideName will cause templates/server-configmap.yaml
  ## to NOT generate a ConfigMap resource
  ##
  configMapOverrideName: ""

  ## Extra labels for Prometheus server ConfigMap (ConfigMap that holds serverFiles)
  extraConfigmapLabels: {}

  ingress:
    ## If true, Prometheus server Ingress will be created
    ##
    enabled: false

    # For Kubernetes >= 1.18 you should specify the ingress-controller via the field ingressClassName
    # See https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/#specifying-the-class-of-an-ingress
    # ingressClassName: nginx

    ## Prometheus server Ingress annotations
    ##
    annotations: {}
    #   kubernetes.io/ingress.class: nginx
    #   kubernetes.io/tls-acme: 'true'

    ## Prometheus server Ingress additional labels
    ##
    extraLabels: {}

    ## Prometheus server Ingress hostnames with optional path
    ## Must be provided if Ingress is enabled
    ##
    hosts: []
    #   - prometheus.domain.com
    #   - domain.com/prometheus

    path: /

    # pathType is only for k8s >= 1.18
    pathType: Prefix

    ## Extra paths to prepend to every host configuration. This is useful when working with annotation based services.
    extraPaths: []
    # - path: /*
    #   backend:
    #     serviceName: ssl-redirect
    #     servicePort: use-annotation

    ## Prometheus server Ingress TLS configuration
    ## Secrets must be manually created in the namespace
    ##
    tls: []
    #   - secretName: prometheus-server-tls
    #     hosts:
    #       - prometheus.domain.com

  ## Server Deployment Strategy type
  strategy:
    type: Recreate

  ## hostAliases allows adding entries to /etc/hosts inside the containers
  hostAliases: []
  #   - ip: "127.0.0.1"
  #     hostnames:
  #       - "example.com"

  ## Node tolerations for server scheduling to nodes with taints
  ## Ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
  ##
  tolerations: []
    # - key: "key"
    #   operator: "Equal|Exists"
    #   value: "value"
    #   effect: "NoSchedule|PreferNoSchedule|NoExecute(1.6 only)"

  ## Node labels for Prometheus server pod assignment
  ## Ref: https://kubernetes.io/docs/user-guide/node-selection/
  ##
  nodeSelector: {}

  ## Pod affinity
  ##
  affinity: {}

  ## Pod topology spread constraints
  ## ref. https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/
  topologySpreadConstraints: []

  ## PodDisruptionBudget settings
  ## ref: https://kubernetes.io/docs/concepts/workloads/pods/disruptions/
  ##
  podDisruptionBudget:
    enabled: false
    maxUnavailable: 1

  ## Use an alternate scheduler, e.g. "stork".
  ## ref: https://kubernetes.io/docs/tasks/administer-cluster/configure-multiple-schedulers/
  ##
  # schedulerName:

  persistentVolume:
    ## If true, Prometheus server will create/use a Persistent Volume Claim
    ## If false, use emptyDir
    ##
    enabled: true

    ## If set it will override the name of the created persistent volume claim
    ## generated by the stateful set.
    ##
    statefulSetNameOverride: ""

    ## Prometheus server data Persistent Volume access modes
    ## Must match those of existing PV or dynamic provisioner
    ## Ref: http://kubernetes.io/docs/user-guide/persistent-volumes/
    ##
    accessModes:
      - ReadWriteOnce

    ## Prometheus server data Persistent Volume labels
    ##
    labels: {}

    ## Prometheus server data Persistent Volume annotations
    ##
    annotations: {}

    ## Prometheus server data Persistent Volume existing claim name
    ## Requires server.persistentVolume.enabled: true
    ## If defined, PVC must be created manually before volume will be bound
    existingClaim: ""

    ## Prometheus server data Persistent Volume mount root path
    ##
    mountPath: /data

    ## Prometheus server data Persistent Volume size
    ##
    size: 8Gi

    ## Prometheus server data Persistent Volume Storage Class
    ## If defined, storageClassName: <storageClass>
    ## If set to "-", storageClassName: "", which disables dynamic provisioning
    ## If undefined (the default) or set to null, no storageClassName spec is
    ##   set, choosing the default provisioner.  (gp2 on AWS, standard on
    ##   GKE, AWS & OpenStack)
    ##
    # storageClass: "-"

    ## Prometheus server data Persistent Volume Binding Mode
    ## If defined, volumeBindingMode: <volumeBindingMode>
    ## If undefined (the default) or set to null, no volumeBindingMode spec is
    ##   set, choosing the default mode.
    ##
    # volumeBindingMode: ""

    ## Subdirectory of Prometheus server data Persistent Volume to mount
    ## Useful if the volume's root directory is not empty
    ##
    subPath: ""

    ## Persistent Volume Claim Selector
    ## Useful if Persistent Volumes have been provisioned in advance
    ## Ref: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#selector
    ##
    # selector:
    #  matchLabels:
    #    release: "stable"
    #  matchExpressions:
    #    - { key: environment, operator: In, values: [ dev ] }

    ## Persistent Volume Name
    ## Useful if Persistent Volumes have been provisioned in advance and you want to use a specific one
    ##
    # volumeName: ""

  emptyDir:
    ## Prometheus server emptyDir volume size limit
    ##
    sizeLimit: ""

  ## Annotations to be added to Prometheus server pods
  ##
  podAnnotations: {}
    # iam.amazonaws.com/role: prometheus

  ## Labels to be added to Prometheus server pods
  ##
  podLabels: {}

  ## Prometheus AlertManager configuration
  ##
  alertmanagers: []

  ## Specify if a Pod Security Policy for node-exporter must be created
  ## Ref: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
  ##
  podSecurityPolicy:
    annotations: {}
      ## Specify pod annotations
      ## Ref: https://kubernetes.io/docs/concepts/policy/pod-security-policy/#apparmor
      ## Ref: https://kubernetes.io/docs/concepts/policy/pod-security-policy/#seccomp
      ## Ref: https://kubernetes.io/docs/concepts/policy/pod-security-policy/#sysctl
      ##
      # seccomp.security.alpha.kubernetes.io/allowedProfileNames: '*'
      # seccomp.security.alpha.kubernetes.io/defaultProfileName: 'docker/default'
      # apparmor.security.beta.kubernetes.io/defaultProfileName: 'runtime/default'

  ## Use a StatefulSet if replicaCount needs to be greater than 1 (see below)
  ##
  replicaCount: 1

  ## Annotations to be added to deployment
  ##
  deploymentAnnotations: {}

  statefulSet:
    ## If true, use a statefulset instead of a deployment for pod management.
    ## This allows to scale replicas to more than 1 pod
    ##
    enabled: false

    annotations: {}
    labels: {}
    podManagementPolicy: OrderedReady

    ## Alertmanager headless service to use for the statefulset
    ##
    headless:
      annotations: {}
      labels: {}
      servicePort: 80
      ## Enable gRPC port on service to allow auto discovery with thanos-querier
      gRPC:
        enabled: false
        servicePort: 10901
        # nodePort: 10901

  ## Prometheus server readiness and liveness probe initial delay and timeout
  ## Ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
  ##
  tcpSocketProbeEnabled: false
  probeScheme: HTTP
  readinessProbeInitialDelay: 30
  readinessProbePeriodSeconds: 5
  readinessProbeTimeout: 4
  readinessProbeFailureThreshold: 3
  readinessProbeSuccessThreshold: 1
  livenessProbeInitialDelay: 30
  livenessProbePeriodSeconds: 15
  livenessProbeTimeout: 10
  livenessProbeFailureThreshold: 3
  livenessProbeSuccessThreshold: 1
  startupProbe:
    enabled: false
    periodSeconds: 5
    failureThreshold: 30
    timeoutSeconds: 10

  ## Prometheus server resource requests and limits
  ## Ref: http://kubernetes.io/docs/user-guide/compute-resources/
  ##
  resources: {}
    # limits:
    #   cpu: 500m
    #   memory: 512Mi
    # requests:
    #   cpu: 500m
    #   memory: 512Mi

  # Required for use in managed kubernetes clusters (such as AWS EKS) with custom CNI (such as calico),
  # because control-plane managed by AWS cannot communicate with pods' IP CIDR and admission webhooks are not working
  ##
  hostNetwork: false

  # When hostNetwork is enabled, this will set to ClusterFirstWithHostNet automatically
  dnsPolicy: ClusterFirst

  # Use hostPort
  # hostPort: 9090

  ## Vertical Pod Autoscaler config
  ## Ref: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler
  verticalAutoscaler:
    ## If true a VPA object will be created for the controller (either StatefulSet or Deployemnt, based on above configs)
    enabled: false
    # updateMode: "Auto"
    # containerPolicies:
    # - containerName: 'prometheus-server'

  # Custom DNS configuration to be added to prometheus server pods
  dnsConfig: {}
    # nameservers:
    #   - 1.2.3.4
    # searches:
    #   - ns1.svc.cluster-domain.example
    #   - my.dns.search.suffix
    # options:
    #   - name: ndots
    #     value: "2"
  #   - name: edns0

  ## Security context to be added to server pods
  ##
  securityContext:
    runAsUser: 65534
    runAsNonRoot: true
    runAsGroup: 65534
    fsGroup: 65534

  ## Security context to be added to server container
  ##
  containerSecurityContext: {}

  service:
    ## If false, no Service will be created for the Prometheus server
    ##
    enabled: true

    annotations: {}
    labels: {}
    clusterIP: ""

    ## List of IP addresses at which the Prometheus server service is available
    ## Ref: https://kubernetes.io/docs/concepts/services-networking/service/#external-ips
    ##
    externalIPs: []

    loadBalancerIP: ""
    loadBalancerSourceRanges: []
    servicePort: 80
    sessionAffinity: None
    type: ClusterIP

    ## Enable gRPC port on service to allow auto discovery with thanos-querier
    gRPC:
      enabled: false
      servicePort: 10901
      # nodePort: 10901

    ## If using a statefulSet (statefulSet.enabled=true), configure the
    ## service to connect to a specific replica to have a consistent view
    ## of the data.
    statefulsetReplica:
      enabled: false
      replica: 0

  ## Prometheus server pod termination grace period
  ##
  terminationGracePeriodSeconds: 300

  ## Prometheus data retention period (default if not specified is 15 days)
  ##
  retention: "15d"

## Prometheus server ConfigMap entries for rule files (allow prometheus labels interpolation)
ruleFiles: {}

## Prometheus server ConfigMap entries
##
serverFiles:
  ## Alerts configuration
  ## Ref: https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
  alerting_rules.yml: {}
  # groups:
  #   - name: Instances
  #     rules:
  #       - alert: InstanceDown
  #         expr: up == 0
  #         for: 5m
  #         labels:
  #           severity: page
  #         annotations:
  #           description: '{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes.'
  #           summary: 'Instance {{ $labels.instance }} down'
  ## DEPRECATED DEFAULT VALUE, unless explicitly naming your files, please use alerting_rules.yml
  alerts: {}

  ## Records configuration
  ## Ref: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
  recording_rules.yml: {}
  ## DEPRECATED DEFAULT VALUE, unless explicitly naming your files, please use recording_rules.yml
  rules: {}

  prometheus.yml:
    rule_files:
      - /etc/config/recording_rules.yml
      - /etc/config/alerting_rules.yml
    ## Below two files are DEPRECATED will be removed from this default values file
      - /etc/config/rules
      - /etc/config/alerts

    scrape_configs:
      - job_name: prometheus
        static_configs:
          - targets:
            - localhost:9090

      # A scrape configuration for running Prometheus on a Kubernetes cluster.
      # This uses separate scrape configs for cluster components (i.e. API server, node)
      # and services to allow each to use different authentication configs.
      #
      # Kubernetes labels will be added as Prometheus labels on metrics via the
      # `labelmap` relabeling action.

      # Scrape config for API servers.
      #
      # Kubernetes exposes API servers as endpoints to the default/kubernetes
      # service so this uses `endpoints` role and uses relabelling to only keep
      # the endpoints associated with the default/kubernetes service using the
      # default named port `https`. This works for single API server deployments as
      # well as HA API server deployments.
      - job_name: 'kubernetes-apiservers'

        kubernetes_sd_configs:
          - role: endpoints

        # Default to scraping over https. If required, just disable this or change to
        # `http`.
        scheme: https

        # This TLS & bearer token file config is used to connect to the actual scrape
        # endpoints for cluster components. This is separate to discovery auth
        # configuration because discovery & scraping are two separate concerns in
        # Prometheus. The discovery auth config is automatic if Prometheus runs inside
        # the cluster. Otherwise, more config options have to be provided within the
        # <kubernetes_sd_config>.
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
          # If your node certificates are self-signed or use a different CA to the
          # master CA, then disable certificate verification below. Note that
          # certificate verification is an integral part of a secure infrastructure
          # so this should only be disabled in a controlled environment. You can
          # disable certificate verification by uncommenting the line below.
          #
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

        # Keep only the default/kubernetes service endpoints for the https port. This
        # will add targets for each API server which Kubernetes adds an endpoint to
        # the default/kubernetes service.
        relabel_configs:
          - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
            action: keep
            regex: default;kubernetes;https

      - job_name: 'kubernetes-nodes'

        # Default to scraping over https. If required, just disable this or change to
        # `http`.
        scheme: https

        # This TLS & bearer token file config is used to connect to the actual scrape
        # endpoints for cluster components. This is separate to discovery auth
        # configuration because discovery & scraping are two separate concerns in
        # Prometheus. The discovery auth config is automatic if Prometheus runs inside
        # the cluster. Otherwise, more config options have to be provided within the
        # <kubernetes_sd_config>.
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
          # If your node certificates are self-signed or use a different CA to the
          # master CA, then disable certificate verification below. Note that
          # certificate verification is an integral part of a secure infrastructure
          # so this should only be disabled in a controlled environment. You can
          # disable certificate verification by uncommenting the line below.
          #
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

        kubernetes_sd_configs:
          - role: node

        relabel_configs:
          - action: labelmap
            regex: __meta_kubernetes_node_label_(.+)
          - target_label: __address__
            replacement: kubernetes.default.svc:443
          - source_labels: [__meta_kubernetes_node_name]
            regex: (.+)
            target_label: __metrics_path__
            replacement: /api/v1/nodes/$1/proxy/metrics


      - job_name: 'kubernetes-nodes-cadvisor'

        # Default to scraping over https. If required, just disable this or change to
        # `http`.
        scheme: https

        # This TLS & bearer token file config is used to connect to the actual scrape
        # endpoints for cluster components. This is separate to discovery auth
        # configuration because discovery & scraping are two separate concerns in
        # Prometheus. The discovery auth config is automatic if Prometheus runs inside
        # the cluster. Otherwise, more config options have to be provided within the
        # <kubernetes_sd_config>.
        tls_config:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
          # If your node certificates are self-signed or use a different CA to the
          # master CA, then disable certificate verification below. Note that
          # certificate verification is an integral part of a secure infrastructure
          # so this should only be disabled in a controlled environment. You can
          # disable certificate verification by uncommenting the line below.
          #
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

        kubernetes_sd_configs:
          - role: node

        # This configuration will work only on kubelet 1.7.3+
        # As the scrape endpoints for cAdvisor have changed
        # if you are using older version you need to change the replacement to
        # replacement: /api/v1/nodes/$1:4194/proxy/metrics
        # more info here https://github.com/coreos/prometheus-operator/issues/633
        relabel_configs:
          - action: labelmap
            regex: __meta_kubernetes_node_label_(.+)
          - target_label: __address__
            replacement: kubernetes.default.svc:443
          - source_labels: [__meta_kubernetes_node_name]
            regex: (.+)
            target_label: __metrics_path__
            replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor

        # Metric relabel configs to apply to samples before ingestion.
        # [Metric Relabeling](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#metric_relabel_configs)
        # metric_relabel_configs:
        # - action: labeldrop
        #   regex: (kubernetes_io_hostname|failure_domain_beta_kubernetes_io_region|beta_kubernetes_io_os|beta_kubernetes_io_arch|beta_kubernetes_io_instance_type|failure_domain_beta_kubernetes_io_zone)

      # Scrape config for service endpoints.
      #
      # The relabeling allows the actual service scrape endpoint to be configured
      # via the following annotations:
      #
      # * `prometheus.io/scrape`: Only scrape services that have a value of
      # `true`, except if `prometheus.io/scrape-slow` is set to `true` as well.
      # * `prometheus.io/scheme`: If the metrics endpoint is secured then you will need
      # to set this to `https` & most likely set the `tls_config` of the scrape config.
      # * `prometheus.io/path`: If the metrics path is not `/metrics` override this.
      # * `prometheus.io/port`: If the metrics are exposed on a different port to the
      # service then set this appropriately.
      # * `prometheus.io/param_<parameter>`: If the metrics endpoint uses parameters
      # then you can set any parameter
      - job_name: 'kubernetes-service-endpoints'
        honor_labels: true

        kubernetes_sd_configs:
          - role: endpoints

        relabel_configs:
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape_slow]
            action: drop
            regex: true
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
            action: replace
            target_label: __scheme__
            regex: (https?)
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
            action: replace
            target_label: __address__
            regex: (.+?)(?::\d+)?;(\d+)
            replacement: $1:$2
          - action: labelmap
            regex: __meta_kubernetes_service_annotation_prometheus_io_param_(.+)
            replacement: __param_$1
          - action: labelmap
            regex: __meta_kubernetes_service_label_(.+)
          - source_labels: [__meta_kubernetes_namespace]
            action: replace
            target_label: namespace
          - source_labels: [__meta_kubernetes_service_name]
            action: replace
            target_label: service
          - source_labels: [__meta_kubernetes_pod_node_name]
            action: replace
            target_label: node

      # Scrape config for slow service endpoints; same as above, but with a larger
      # timeout and a larger interval
      #
      # The relabeling allows the actual service scrape endpoint to be configured
      # via the following annotations:
      #
      # * `prometheus.io/scrape-slow`: Only scrape services that have a value of `true`
      # * `prometheus.io/scheme`: If the metrics endpoint is secured then you will need
      # to set this to `https` & most likely set the `tls_config` of the scrape config.
      # * `prometheus.io/path`: If the metrics path is not `/metrics` override this.
      # * `prometheus.io/port`: If the metrics are exposed on a different port to the
      # service then set this appropriately.
      # * `prometheus.io/param_<parameter>`: If the metrics endpoint uses parameters
      # then you can set any parameter
      - job_name: 'kubernetes-service-endpoints-slow'
        honor_labels: true

        scrape_interval: 5m
        scrape_timeout: 30s

        kubernetes_sd_configs:
          - role: endpoints

        relabel_configs:
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape_slow]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
            action: replace
            target_label: __scheme__
            regex: (https?)
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
            action: replace
            target_label: __address__
            regex: (.+?)(?::\d+)?;(\d+)
            replacement: $1:$2
          - action: labelmap
            regex: __meta_kubernetes_service_annotation_prometheus_io_param_(.+)
            replacement: __param_$1
          - action: labelmap
            regex: __meta_kubernetes_service_label_(.+)
          - source_labels: [__meta_kubernetes_namespace]
            action: replace
            target_label: namespace
          - source_labels: [__meta_kubernetes_service_name]
            action: replace
            target_label: service
          - source_labels: [__meta_kubernetes_pod_node_name]
            action: replace
            target_label: node

      - job_name: 'prometheus-pushgateway'
        honor_labels: true

        kubernetes_sd_configs:
          - role: service

        relabel_configs:
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
            action: keep
            regex: pushgateway

      # Example scrape config for probing services via the Blackbox Exporter.
      #
      # The relabeling allows the actual service scrape endpoint to be configured
      # via the following annotations:
      #
      # * `prometheus.io/probe`: Only probe services that have a value of `true`
      - job_name: 'kubernetes-services'
        honor_labels: true

        metrics_path: /probe
        params:
          module: [http_2xx]

        kubernetes_sd_configs:
          - role: service

        relabel_configs:
          - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_probe]
            action: keep
            regex: true
          - source_labels: [__address__]
            target_label: __param_target
          - target_label: __address__
            replacement: blackbox
          - source_labels: [__param_target]
            target_label: instance
          - action: labelmap
            regex: __meta_kubernetes_service_label_(.+)
          - source_labels: [__meta_kubernetes_namespace]
            target_label: namespace
          - source_labels: [__meta_kubernetes_service_name]
            target_label: service

      # Example scrape config for pods
      #
      # The relabeling allows the actual pod scrape endpoint to be configured via the
      # following annotations:
      #
      # * `prometheus.io/scrape`: Only scrape pods that have a value of `true`,
      # except if `prometheus.io/scrape-slow` is set to `true` as well.
      # * `prometheus.io/scheme`: If the metrics endpoint is secured then you will need
      # to set this to `https` & most likely set the `tls_config` of the scrape config.
      # * `prometheus.io/path`: If the metrics path is not `/metrics` override this.
      # * `prometheus.io/port`: Scrape the pod on the indicated port instead of the default of `9102`.
      - job_name: 'kubernetes-pods'
        honor_labels: true

        kubernetes_sd_configs:
          - role: pod

        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_slow]
            action: drop
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scheme]
            action: replace
            regex: (https?)
            target_label: __scheme__
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
            action: replace
            regex: (\d+);(([A-Fa-f0-9]{1,4}::?){1,7}[A-Fa-f0-9]{1,4})
            replacement: '[$2]:$1'
            target_label: __address__
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
            action: replace
            regex: (\d+);((([0-9]+?)(\.|$)){4})
            replacement: $2:$1
            target_label: __address__
          - action: labelmap
            regex: __meta_kubernetes_pod_annotation_prometheus_io_param_(.+)
            replacement: __param_$1
          - action: labelmap
            regex: __meta_kubernetes_pod_label_(.+)
          - source_labels: [__meta_kubernetes_namespace]
            action: replace
            target_label: namespace
          - source_labels: [__meta_kubernetes_pod_name]
            action: replace
            target_label: pod
          - source_labels: [__meta_kubernetes_pod_phase]
            regex: Pending|Succeeded|Failed|Completed
            action: drop

      # Example Scrape config for pods which should be scraped slower. An useful example
      # would be stackriver-exporter which queries an API on every scrape of the pod
      #
      # The relabeling allows the actual pod scrape endpoint to be configured via the
      # following annotations:
      #
      # * `prometheus.io/scrape-slow`: Only scrape pods that have a value of `true`
      # * `prometheus.io/scheme`: If the metrics endpoint is secured then you will need
      # to set this to `https` & most likely set the `tls_config` of the scrape config.
      # * `prometheus.io/path`: If the metrics path is not `/metrics` override this.
      # * `prometheus.io/port`: Scrape the pod on the indicated port instead of the default of `9102`.
      - job_name: 'kubernetes-pods-slow'
        honor_labels: true

        scrape_interval: 5m
        scrape_timeout: 30s

        kubernetes_sd_configs:
          - role: pod

        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_slow]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scheme]
            action: replace
            regex: (https?)
            target_label: __scheme__
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
            action: replace
            regex: (\d+);(([A-Fa-f0-9]{1,4}::?){1,7}[A-Fa-f0-9]{1,4})
            replacement: '[$2]:$1'
            target_label: __address__
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
            action: replace
            regex: (\d+);((([0-9]+?)(\.|$)){4})
            replacement: $2:$1
            target_label: __address__
          - action: labelmap
            regex: __meta_kubernetes_pod_annotation_prometheus_io_param_(.+)
            replacement: __param_$1
          - action: labelmap
            regex: __meta_kubernetes_pod_label_(.+)
          - source_labels: [__meta_kubernetes_namespace]
            action: replace
            target_label: namespace
          - source_labels: [__meta_kubernetes_pod_name]
            action: replace
            target_label: pod
          - source_labels: [__meta_kubernetes_pod_phase]
            regex: Pending|Succeeded|Failed|Completed
            action: drop

# adds additional scrape configs to prometheus.yml
# must be a string so you have to add a | after extraScrapeConfigs:
# example adds prometheus-blackbox-exporter scrape config
extraScrapeConfigs: ""
  # - job_name: 'prometheus-blackbox-exporter'
  #   metrics_path: /probe
  #   params:
  #     module: [http_2xx]
  #   static_configs:
  #     - targets:
  #       - https://example.com
  #   relabel_configs:
  #     - source_labels: [__address__]
  #       target_label: __param_target
  #     - source_labels: [__param_target]
  #       target_label: instance
  #     - target_label: __address__
  #       replacement: prometheus-blackbox-exporter:9115

# Adds option to add alert_relabel_configs to avoid duplicate alerts in alertmanager
# useful in H/A prometheus with different external labels but the same alerts
alertRelabelConfigs: {}
  # alert_relabel_configs:
  # - source_labels: [dc]
  #   regex: (.+)\d+
  #   target_label: dc

networkPolicy:
  ## Enable creation of NetworkPolicy resources.
  ##
  enabled: false

# Force namespace of namespaced resources
forceNamespace: ""

# Extra manifests to deploy as an array
extraManifests: []
  # - |
  #   apiVersion: v1
  #   kind: ConfigMap
  #   metadata:
  #   labels:
  #     name: prometheus-extra
  #   data:
  #     extra-data: "value"

# Configuration of subcharts defined in Chart.yaml

## alertmanager sub-chart configurable values
## Please see https://github.com/prometheus-community/helm-charts/tree/main/charts/alertmanager
##
alertmanager:
  ## If false, alertmanager will not be installed
  ##
  enabled: true

  persistence:
    size: 2Gi

  podSecurityContext:
    runAsUser: 65534
    runAsNonRoot: true
    runAsGroup: 65534
    fsGroup: 65534

## kube-state-metrics sub-chart configurable values
## Please see https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics
##
kube-state-metrics:
  ## If false, kube-state-metrics sub-chart will not be installed
  ##
  enabled: true

## promtheus-node-exporter sub-chart configurable values
## Please see https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-node-exporter
##
prometheus-node-exporter:
  ## If false, node-exporter will not be installed
  ##
  enabled: true

  rbac:
    pspEnabled: false

  containerSecurityContext:
    allowPrivilegeEscalation: false

## pprometheus-pushgateway sub-chart configurable values
## Please see https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-pushgateway
##
prometheus-pushgateway:
  ## If false, pushgateway will not be installed
  ##
  enabled: true

  # Optional service annotations
  serviceAnnotations:
    prometheus.io/probe: pushgateway

