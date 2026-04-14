{{/*
  Plugin volume mount definition
*/}}
{{- define "vcluster.plugins.volumeMounts" -}}
{{- $state := dict "pluginFound" false -}}
{{- range $key, $container := .Values.plugin }}
{{- if and (eq $container.version "v2") $container.image }}
{{- $_ := set $state "pluginFound" true -}}
{{- end }}
{{- end }}
{{- if $state.pluginFound }}
- mountPath: /plugins
  name: plugins
{{- else }}
{{- $pluginsState := dict "pluginFound" false -}}
{{- range $key, $container := .Values.plugins }}
{{- if $container.image }}
{{- $_ := set $pluginsState "pluginFound" true -}}
{{- end }}
{{- end }}
{{- if $pluginsState.pluginFound }}
- mountPath: /plugins
  name: plugins
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Plugin volume definition
*/}}
{{- define "vcluster.plugins.volumes" -}}
{{- $state := dict "pluginFound" false -}}
{{- range $key, $container := .Values.plugin }}
{{- if and (eq $container.version "v2") $container.image }}
{{- $_ := set $state "pluginFound" true -}}
{{- end }}
{{- end }}
{{- if $state.pluginFound }}
- name: plugins
  emptyDir: {}
{{- else }}
{{- $pluginsState := dict "pluginFound" false -}}
{{- range $key, $container := .Values.plugins }}
{{- if $container.image }}
{{- $_ := set $pluginsState "pluginFound" true -}}
{{- end }}
{{- end }}
{{- if $pluginsState.pluginFound }}
- name: plugins
  emptyDir: {}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Sidecar container definition for the legacy syncer parts
*/}}
{{- define "vcluster.legacyPlugins.containers" -}}
{{- $counter := -1 -}}
{{- range $key, $container := .Values.plugin }}
{{- if ne $container.version "v2" }}
{{- $counter = add1 $counter }}
- {{- if $.Values.controlPlane.advanced.defaultImageRegistry }}
  image: {{ $.Values.controlPlane.advanced.defaultImageRegistry }}/{{ $container.image }}
  {{- else }}
  image: {{ $container.image }}
  {{- end }}
  {{- if $container.name }}
  name: {{ $container.name | quote }}
  {{- else }}
  name: {{ $key | quote }}
  {{- end }}
  {{- if $container.imagePullPolicy }}
  imagePullPolicy: {{ $container.imagePullPolicy }}
  {{- end }}
  {{- if $container.workingDir }}
  workingDir: {{ $container.workingDir }}
  {{- end }}
  {{- if $container.command }}
  command:
    {{- range $commandIndex, $command := $container.command }}
    - {{ $command | quote }}
    {{- end }}
  {{- end }}
  {{- if $container.args }}
  args:
    {{- range $argIndex, $arg := $container.args }}
    - {{ $arg | quote }}
    {{- end }}
  {{- end }}
  {{- if $container.terminationMessagePath }}
  terminationMessagePath: {{ $container.terminationMessagePath }}
  {{- end }}
  {{- if $container.terminationMessagePolicy }}
  terminationMessagePolicy: {{ $container.terminationMessagePolicy }}
  {{- end }}
  env:
    - name: VCLUSTER_PLUGIN_ADDRESS
      value: "localhost:{{ add 14000 $counter }}"
    - name: VCLUSTER_PLUGIN_NAME
      value: "{{ $key }}"
  {{- if $container.env }}
{{ toYaml $container.env | indent 4 }}
  {{- end }}
  envFrom:
{{ toYaml $container.envFrom | indent 4 }}
  securityContext:
{{ toYaml $container.securityContext | indent 4 }}
  lifecycle:
{{ toYaml $container.lifecycle | indent 4 }}
  livenessProbe:
{{ toYaml $container.livenessProbe | indent 4 }}
  readinessProbe:
{{ toYaml $container.readinessProbe | indent 4 }}
  startupProbe:
{{ toYaml $container.startupProbe | indent 4 }}
  volumeDevices:
{{ toYaml $container.volumeDevices | indent 4 }}
  volumeMounts:
{{ toYaml $container.volumeMounts | indent 4 }}
  {{- if $container.resources }}
  resources:
{{ toYaml $container.resources | indent 4 }}
  {{- end }}
{{- end }}
{{- end }}
{{- end -}}
