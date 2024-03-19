{{/*
  ControlPlane Kind
*/}}
{{- define "vcluster.kind" -}}
{{ if include "vcluster.persistence.volumeClaim.enabled" . }}StatefulSet{{ else }}Deployment{{ end }}
{{- end -}}

{{/*
  StatefulSet Persistence Options
*/}}
{{- define "vcluster.persistence" -}}
{{- if .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates }}
volumeClaimTemplates:
{{ toYaml .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates | indent 2 }}
{{- else if include "vcluster.persistence.volumeClaim.enabled" . }}
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
{{- if .Values.controlPlane.statefulSet.persistence.volumeClaimTemplates -}}
{{- true -}}
{{- else if and (not .Values.controlPlane.statefulSet.persistence.volumeClaim.disabled) (not (include "vcluster.externalEtcd.enabled" .)) -}}
{{- true -}}
{{- end -}}
{{- end -}}
