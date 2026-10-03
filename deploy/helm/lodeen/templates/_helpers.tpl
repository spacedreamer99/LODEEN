{{/* Expand the name of the chart. */}}
{{- define "lodeen.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Fully qualified app name. */}}
{{- define "lodeen.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/* Chart label. */}}
{{- define "lodeen.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Common labels. */}}
{{- define "lodeen.labels" -}}
helm.sh/chart: {{ include "lodeen.chart" . }}
{{ include "lodeen.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Selector labels. */}}
{{- define "lodeen.selectorLabels" -}}
app.kubernetes.io/name: {{ include "lodeen.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ include "lodeen.name" . }}
{{- end }}

{{/* ServiceAccount name. */}}
{{- define "lodeen.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "lodeen.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
