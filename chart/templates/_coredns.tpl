{{/*
  Define a common coredns config
*/}}
{{- define "vcluster.corefile" -}}
Corefile: |-
  {{- if .Values.controlPlane.coredns.overwriteConfig }}
{{ .Values.controlPlane.coredns.overwriteConfig | indent 8 }}
  {{- else }}
  .:1053 {
      errors
      health
      ready
      {{- if and .Values.controlPlane.coredns.embedded .Values.networking.resolveDNS }}
      vcluster
      {{- end }}
      {{- if .Values.networking.advanced.proxyKubelets.byHostname }}
      rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
      {{- end }}
      kubernetes{{ if and (.Values.networking.advanced.clusterDomain) (ne .Values.networking.advanced.clusterDomain "cluster.local") }} {{ .Values.networking.advanced.clusterDomain }}{{ end }} cluster.local in-addr.arpa ip6.arpa {
          {{- if .Values.controlPlane.coredns.embedded }}
          kubeconfig /data/vcluster/admin.conf
          {{- end }}
          pods insecure
          {{- if .Values.networking.advanced.fallbackHostCluster }}
          fallthrough cluster.local in-addr.arpa ip6.arpa
          {{- else }}
          fallthrough in-addr.arpa ip6.arpa
          {{- end }}
      }
      hosts /etc/NodeHosts {
          ttl 60
          reload 15s
          fallthrough
      }
      prometheus :9153
      {{- if .Values.networking.advanced.fallbackHostCluster }}
      forward . {{`{{.HOST_CLUSTER_DNS}}`}}
      {{- else if .Values.policies.networkPolicy.enabled }}
      forward . /etc/resolv.conf {{ .Values.policies.networkPolicy.fallbackDns }} {
          policy sequential
      }
      {{- else }}
      forward . /etc/resolv.conf
      {{- end }}
      cache 30
      loop
      loadbalance
  }

  import /etc/coredns/custom/*.server
  {{- end }}
{{- end -}}
