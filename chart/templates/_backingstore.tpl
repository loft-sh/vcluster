{{/*
  is external etcd enabled?
*/}}
{{- define "vcluster.externalEtcd.enabled" -}}
{{- if and (eq (include "vcluster.distro" .) "k8s") (not .Values.controlPlane.backingStore.embeddedEtcd.enabled) -}}
{{- true -}}
{{- else if and (eq (include "vcluster.distro" .) "eks") (not .Values.controlPlane.backingStore.embeddedEtcd.enabled) -}}
{{- true -}}
{{- else if .Values.controlPlane.backingStore.externalEtcd.enabled -}}
{{- true -}}
{{- end -}}
{{- end -}}

{{/*
  migrate from external etcd?
*/}}
{{- define "vcluster.externalEtcd.migrate" -}}
{{- if and .Values.controlPlane.backingStore.embeddedEtcd.enabled .Values.controlPlane.backingStore.embeddedEtcd.migrateFromExternalEtcd -}}
{{- true -}}
{{- end -}}
{{- end -}}

