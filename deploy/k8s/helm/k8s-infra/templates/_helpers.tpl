{{/*
Common labels for k8s-infra resources
*/}}
{{- define "k8s-infra.labels" -}}
app.kubernetes.io/name: k8s-infra
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: how-dev-iam
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
