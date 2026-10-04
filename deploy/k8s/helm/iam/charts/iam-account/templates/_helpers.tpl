{{/*
Common labels for iam-account
*/}}
{{- define "iam-account.labels" -}}
app.kubernetes.io/name: iam-account
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: account
app.kubernetes.io/part-of: how-dev-iam
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "iam-account.selectorLabels" -}}
app.kubernetes.io/name: iam-account
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Fullname / resource name
*/}}
{{- define "iam-account.fullname" -}}
{{- printf "iam-account" -}}
{{- end }}

{{/*
Image: <registry>/<repository>/iam-account:<tag>
  - repository 优先取子 chart values.image.repository（完整路径）
  - 否则拼 global.image.registry + "/" + global.image.repository + "/iam-account"
  - tag 优先取子 chart values.image.tag；否则回退 global.image.tag；再回退 .Chart.AppVersion
*/}}
{{- define "iam-account.image" -}}
{{- $repo := .Values.image.repository -}}
{{- if not $repo -}}
  {{- $registry := .Values.global.image.registry | default "" -}}
  {{- $orgRepo  := .Values.global.image.repository | default "" -}}
  {{- if and $registry $orgRepo -}}
    {{- $repo = printf "%s/%s/iam-account" $registry $orgRepo -}}
  {{- else if $registry -}}
    {{- $repo = printf "%s/iam-account" $registry -}}
  {{- else -}}
    {{- $repo = "iam-account" -}}
  {{- end -}}
{{- end -}}
{{- $tag := .Values.image.tag | default .Values.global.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end }}

{{/*
选出实际使用的 database 名：优先子 chart values.database，回退 global.config.mysql.database
*/}}
{{- define "iam-account.database" -}}
{{- default .Values.global.config.mysql.database .Values.database -}}
{{- end }}

{{/*
拼接 MySQL DSN，符合 go-sql-driver/mysql 与 iam-account/config.go 要求。
*/}}
{{- define "iam-account.mysqlDSN" -}}
{{- $m := .Values.global.config.mysql -}}
{{- $db := include "iam-account.database" . -}}
{{- printf "%s:%s@tcp(%s:%v)/%s?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s" $m.user $m.password $m.host $m.port $db -}}
{{- end }}
