{{/*
Common labels
*/}}
{{- define "init-jobs.labels" -}}
app.kubernetes.io/name: init-jobs
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: how-dev-iam
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
mysql-schema Job / ConfigMap 名称
用法：{{ include "init-jobs.mysqlSchemaName" (dict "root" $ "service" $svc.name) }}
*/}}
{{- define "init-jobs.mysqlSchemaName" -}}
{{- printf "init-mysql-schema-%s" .service | trunc 63 | trimSuffix "-" -}}
{{- end }}
