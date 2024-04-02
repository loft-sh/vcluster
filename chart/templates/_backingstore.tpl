{{/*
  is deploy etcd enabled?
*/}}
{{- define "vcluster.database.embedded.enabled" -}}
{{- $backingStores := 0 -}}
{{- if .Values.controlPlane.backingStore.etcd.embedded.enabled -}}
{{- $backingStores = add1 $backingStores -}}
{{- end -}}
{{- if .Values.controlPlane.backingStore.etcd.deploy.enabled -}}
{{- $backingStores = add1 $backingStores -}}
{{- end -}}
{{- if .Values.controlPlane.backingStore.database.embedded.enabled -}}
{{- $backingStores = add1 $backingStores -}}
{{- end -}}
{{- if .Values.controlPlane.backingStore.database.external.enabled -}}
{{- $backingStores = add1 $backingStores -}}
{{- end -}}
{{- if gt $backingStores 1 -}}
{{- fail "you can only enable one backingStore at the same time" -}}
{{- else if or (eq $backingStores 0) .Values.controlPlane.backingStore.database.embedded.enabled -}}
{{- true -}}
{{- end -}}
{{- end -}}

{{/*
  migrate from deployed etcd?
*/}}
{{- define "vcluster.etcd.embedded.migrate" -}}
{{- if and .Values.controlPlane.backingStore.etcd.embedded.enabled .Values.controlPlane.backingStore.etcd.embedded.migrateFromDeployedEtcd -}}
{{- true -}}
{{- end -}}
{{- end -}}

