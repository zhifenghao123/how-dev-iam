{{/*
Common labels for mysql sub-chart
*/}}
{{- define "mysql.labels" -}}
app.kubernetes.io/name: mysql
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: database
app.kubernetes.io/part-of: how-dev-iam-middleware
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mysql.selectorLabels" -}}
app.kubernetes.io/name: mysql
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Fully qualified service / workload name.
统一命名前缀由 umbrella chart 的 global.namePrefix 提供，默认 "middleware"。
*/}}
{{- define "mysql.fullname" -}}
{{- $prefix := default "middleware" .Values.global.namePrefix -}}
{{- printf "%s-mysql" $prefix | trunc 63 | trimSuffix "-" -}}
{{- end }}
