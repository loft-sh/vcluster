{{/*
  Fail the install/upgrade if any volume-snapshot value is set.
  These were removed in 0.36.0. The config fields are retained as no-ops so
  existing configs still parse, but the chart rejects them so users notice.
*/}}
{{- define "vcluster.legacy.volumeSnapshots.validate" }}
{{- $sync := .Values.sync | default dict }}
{{- $syncToHost := $sync.toHost | default dict }}
{{- $syncFromHost := $sync.fromHost | default dict }}
{{- $deploy := .Values.deploy | default dict }}
{{- $rbac := .Values.rbac | default dict }}
{{- if hasKey $syncToHost "volumeSnapshots" }}
{{- fail "sync.toHost.volumeSnapshots was removed in 0.36.0 and is no longer supported. Please remove it from your values." }}
{{- end }}
{{- if hasKey $syncToHost "volumeSnapshotContents" }}
{{- fail "sync.toHost.volumeSnapshotContents was removed in 0.36.0 and is no longer supported. Please remove it from your values." }}
{{- end }}
{{- if hasKey $syncFromHost "volumeSnapshotClasses" }}
{{- fail "sync.fromHost.volumeSnapshotClasses was removed in 0.36.0 and is no longer supported. Please remove it from your values." }}
{{- end }}
{{- if hasKey $deploy "volumeSnapshotController" }}
{{- fail "deploy.volumeSnapshotController was removed in 0.36.0 and is no longer supported. Please remove it from your values." }}
{{- end }}
{{- if hasKey $rbac "enableVolumeSnapshotRules" }}
{{- fail "rbac.enableVolumeSnapshotRules was removed in 0.36.0 and is no longer supported. Please remove it from your values." }}
{{- end }}
{{- end }}