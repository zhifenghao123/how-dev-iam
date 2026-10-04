{{/*
Common labels for iam-front
*/}}
{{- define "iam-front.labels" -}}
app.kubernetes.io/name: iam-front
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: frontend
app.kubernetes.io/part-of: how-dev-iam
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "iam-front.selectorLabels" -}}
app.kubernetes.io/name: iam-front
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Fullname / resource name
*/}}
{{- define "iam-front.fullname" -}}
{{- printf "iam-front" -}}
{{- end }}

{{/*
Image: <registry>/<repository>/iam-front:<tag>
*/}}
{{- define "iam-front.image" -}}
{{- $repo := .Values.image.repository -}}
{{- if not $repo -}}
  {{- $registry := .Values.global.image.registry | default "" -}}
  {{- $orgRepo  := .Values.global.image.repository | default "" -}}
  {{- if and $registry $orgRepo -}}
    {{- $repo = printf "%s/%s/iam-front" $registry $orgRepo -}}
  {{- else if $registry -}}
    {{- $repo = printf "%s/iam-front" $registry -}}
  {{- else -}}
    {{- $repo = "iam-front" -}}
  {{- end -}}
{{- end -}}
{{- $tag := .Values.image.tag | default .Values.global.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end }}
