{{/*
  Define a common coredns config
*/}}
{{- define "vcluster.corefile" -}}
Corefile: |-
  {{- if .Values.coredns.config }}
{{ .Values.coredns.config | indent 8 }}
  {{- else }}
  .:1053 {
      errors
      health
      ready
      rewrite name regex .*\.nodes\.vcluster\.com kubernetes.default.svc.cluster.local
      kubernetes cluster.local in-addr.arpa ip6.arpa {
          {{- if .Values.pro }}
          {{- if .Values.coredns.integrated }}
          kubeconfig /pki/admin.conf
          {{- end }}
          {{- end }}
          pods insecure
          {{- if .Values.fallbackHostDns }}
          fallthrough cluster.local in-addr.arpa ip6.arpa
          {{- else }}
          fallthrough in-addr.arpa ip6.arpa
          {{- end }}
      }
      {{- if and .Values.coredns.integrated .Values.coredns.plugin.enabled }}
      vcluster {{ toYaml .Values.coredns.plugin.config | b64enc }}
      {{- end }}
      hosts /etc/NodeHosts {
          ttl 60
          reload 15s
          fallthrough
      }
      prometheus :9153
      {{- if .Values.fallbackHostDns }}
      forward . {{`{{.HOST_CLUSTER_DNS}}`}}
      {{- else if and .Values.isolation.enabled  .Values.isolation.networkPolicy.enabled }}
      forward . /etc/resolv.conf {{ .Values.coredns.fallback }} {
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
