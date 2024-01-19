{{/*
  Plugin config definition
*/}}
{{- define "vcluster.plugins.config" -}}
{{- $pluginFound := false -}}
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.config) }}
{{- continue }}
{{- end }}
{{- $pluginFound = true -}}
{{- end }}
{{- if $pluginFound }}
- name: PLUGIN_CONFIG
  value: |-
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.config) }}
{{- continue }}
{{- end }}
    {{ $key }}: {{ toYaml $container.config | nindent 6 }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Plugin volume mount definition
*/}}
{{- define "vcluster.plugins.volumeMounts" -}}
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.image) }}
{{- continue }}
{{- end }}
- mountPath: /plugins
  name: plugins
{{- break }}
{{- end }}
{{- end -}}

{{/*
  Plugin volume definition
*/}}
{{- define "vcluster.plugins.volumes" -}}
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.image) }}
{{- continue }}
{{- end }}
- name: plugins
  emptyDir: {}
{{- break }}
{{- end }}
{{- end -}}

{{/*
  Plugin init container definition
*/}}
{{- define "vcluster.plugins.initContainers" -}}
{{- range $key, $container := .Values.plugin }}
{{- if or (ne $container.version "v2") (not $container.image) }}
{{- continue }}
{{- end }}
- image: {{ $.Values.defaultImageRegistry }}{{ $container.image }}
  {{- if $container.name }}
  name: {{ $container.name | quote }}
  {{- else }}
  name: {{ $key | quote }}
  {{- end }}
  {{- if $container.imagePullPolicy }}
  imagePullPolicy: {{ $container.imagePullPolicy }}
  {{- end }}
  {{- if or $container.command $container.args }}
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
  {{- else }}
  command: ["sh"]
  args: ["-c", "cp -r /plugin /plugins/{{ $key }}"]
  {{- end }}
  securityContext:
{{ toYaml $container.securityContext | indent 4 }}
  {{- if $container.volumeMounts }}
  volumeMounts:
{{ toYaml $container.volumeMounts | indent 4 }}
  {{- else }}
  volumeMounts:
    - mountPath: /plugins
      name: plugins
  {{- end }}
  {{- if $container.resources }}
  resources:
{{ toYaml $container.resources | indent 4 }}
  {{- end }}
{{- end }}
{{- end -}}

{{/*
  Extra Syncer Args for the legacy Plugins
*/}}
{{- define "vcluster.legacyPlugins.args" -}}
{{- range $key, $container := .Values.plugin }}
{{- if eq $container.version "v2" }}
{{- continue }}
{{- end }}
{{- if not $container.optional }}
- --plugins={{ $key }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
  Sidecar container definition for the legacy syncer parts
*/}}
{{- define "vcluster.legacyPlugins.containers" -}}
{{- $counter := -1 -}}
{{- range $key, $container := .Values.plugin }}
{{- if eq $container.version "v2" }}
{{ continue }}
{{- end }}
{{- $counter = add1 $counter }}
- image: {{ $.Values.defaultImageRegistry }}{{ $container.image }}
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


