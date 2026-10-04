{{/*
Common labels for redis sub-chart
*/}}
{{- define "redis.labels" -}}
app.kubernetes.io/name: redis
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: cache
app.kubernetes.io/part-of: how-dev-iam-middleware
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "redis.selectorLabels" -}}
app.kubernetes.io/name: redis
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Fully qualified service / workload name.
*/}}
{{- define "redis.fullname" -}}
{{- $prefix := default "middleware" .Values.global.namePrefix -}}
{{- printf "%s-redis" $prefix | trunc 63 | trimSuffix "-" -}}
{{- end }}
