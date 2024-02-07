{{/*
  deployment kind
*/}}
{{- define "vcluster.kind" -}}
{{ if and .Values.embeddedEtcd.enabled .Values.pro }}StatefulSet{{ else }}Deployment{{ end }}
{{- end -}}

{{/*
  service name for statefulset
*/}}
{{- define "vcluster.statefulset.serviceName" }}
{{- if .Values.embeddedEtcd.enabled }}
serviceName: {{ .Release.Name }}-headless
{{- end }}
{{- end -}}

{{/*
  volumeClaimTemplate
*/}}
{{- define "vcluster.statefulset.volumeClaimTemplate" }}
{{- if .Values.embeddedEtcd.enabled }}
{{- if .Values.autoDeletePersistentVolumeClaims }}
{{- if ge (int .Capabilities.KubeVersion.Minor) 27 }}
persistentVolumeClaimRetentionPolicy:
  whenDeleted: Delete
{{- end }}
{{- end }}
{{- if (hasKey .Values "volumeClaimTemplates") }}
volumeClaimTemplates:
{{ toYaml .Values.volumeClaimTemplates | indent 4 }}
{{- else if .Values.syncer.storage.persistence }}
volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      {{- if .Values.syncer.storage.className }}
      storageClassName: {{ .Values.syncer.storage.className }}
      {{- end }}
      resources:
        requests:
          storage: {{ .Values.syncer.storage.size }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  deployment strategy
*/}}
{{- define "vcluster.deployment.strategy" }}
{{- if not .Values.embeddedEtcd.enabled }}
strategy:
  rollingUpdate:
    maxSurge: 1
    {{- if (eq (int .Values.syncer.replicas) 1) }}
    maxUnavailable: 0
    {{- else }}
    maxUnavailable: 1
    {{- end }}
  type: RollingUpdate
{{- end }}
{{- end -}}
