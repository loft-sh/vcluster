{{/*
  Fail the install/upgrade if the kubevirt CSI driver is enabled without private nodes. The control plane provisions
  the volumes within its own namespace and hot plugs them into the virtual machines that back the nodes, which only
  exists in private nodes mode. The storage class of the driver also cannot be the default storage class next to the
  one of the local path provisioner, because two default storage classes leave it to Kubernetes which provisioner a
  claim without a storage class ends up on.
*/}}
{{- define "vcluster.kubeVirtCSI.validate" }}
{{- if .Values.deploy.csi.kubeVirt.enabled }}
{{- if not .Values.privateNodes.enabled }}
{{- fail "deploy.csi.kubeVirt is only supported in private nodes mode, please set privateNodes.enabled=true" }}
{{- end }}
{{- $storageClass := .Values.deploy.csi.kubeVirt.storageClass | default dict }}
{{- if and $storageClass.enabled $storageClass.default .Values.deploy.localPathProvisioner.enabled }}
{{- fail "deploy.csi.kubeVirt.storageClass.default and deploy.localPathProvisioner both create a default storage class, please disable deploy.localPathProvisioner or set deploy.csi.kubeVirt.storageClass.default=false" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
  The chart owns the vcluster-logs volume while file logging is enabled. Reject a
  user-provided volume with that name instead of rendering an invalid pod spec or
  silently changing the storage and size-limit guarantees of managed file logging.
*/}}
{{- define "vcluster.fileLogging.validate" }}
{{- if include "vcluster.fileLoggingEnabled" . }}
{{- range .Values.controlPlane.statefulSet.persistence.addVolumes }}
{{- if eq (default "" .name) "vcluster-logs" }}
{{- fail "controlPlane.statefulSet.persistence.addVolumes must not define vcluster-logs when logging.file.enabled is true; the chart reserves this volume" }}
{{- end }}
{{- end }}
{{- range .Values.controlPlane.statefulSet.persistence.addVolumeMounts }}
{{- if eq (default "" .name) "vcluster-logs" }}
{{- fail "controlPlane.statefulSet.persistence.addVolumeMounts must not define vcluster-logs when logging.file.enabled is true; the chart reserves this volume" }}
{{- end }}
{{- end }}
{{- end }}
{{- range .Values.controlPlane.statefulSet.env }}
{{- if eq (default "" .name) "LOFT_LOG_FILE" }}
{{- fail "controlPlane.statefulSet.env must not set LOFT_LOG_FILE; the log file path is fixed and controlled by logging.file.enabled" }}
{{- end }}
{{- end }}
{{- range .Values.controlPlane.statefulSet.sidecarContainers }}
{{- if or (eq (default "" .name) "syncer") (eq (default "" .name) "kubernetes") }}
{{- fail (printf "controlPlane.statefulSet.sidecarContainers must not use the reserved name %q" .name) }}
{{- end }}
{{- end }}
{{- range .Values.controlPlane.statefulSet.initContainers }}
{{- if or (eq (default "" .name) "syncer") (eq (default "" .name) "kubernetes") }}
{{- fail (printf "controlPlane.statefulSet.initContainers must not use the reserved name %q" .name) }}
{{- end }}
{{- end }}
{{- end }}

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
