{{/*
  StatefulSet Persistence Options
*/}}
{{- define "vcluster.persistence" -}}
{{- if and .Values.controlPlane.backingStore.embeddedEtcd.enabled (include "vcluster.externalEtcd.enabled" .) -}}
{{- fail "embeddedEtcd and externalEtcd cannot be enabled at the same time together" }}
{{- end -}}
{{- if .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates }}
{{- if ge (int .Capabilities.KubeVersion.Minor) 27 }}
persistentVolumeClaimRetentionPolicy:
  whenDeleted: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.retentionPolicy }}
{{- end }}
volumeClaimTemplates:
{{ toYaml .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates | indent 2 }}
{{- else if include "vcluster.persistence.volumeClaim.enabled" . }}
{{- if ge (int .Capabilities.KubeVersion.Minor) 27 }}
persistentVolumeClaimRetentionPolicy:
  whenDeleted: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.retentionPolicy }}
{{- end }}
volumeClaimTemplates:
- metadata:
    name: data
  spec:
    accessModes: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.accessModes }}
    {{- if .Values.controlPlane.statefulSet.persistence.volumeClaim.storageClass }}
    storageClassName: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.storageClass }}
    {{- end }}
    resources:
      requests:
        storage: {{ .Values.controlPlane.statefulSet.persistence.volumeClaim.size }}
{{- end }}
{{- end -}}

{{/*
  is persistence enabled?
*/}}
{{- define "vcluster.persistence.volumeClaim.enabled" -}}
{{- if and (not .Values.controlPlane.statefulSet.persistence.volumeClaim.disabled) (not (include "vcluster.externalEtcd.enabled" .)) -}}
{{- true -}}
{{- end -}}
{{- end -}}
