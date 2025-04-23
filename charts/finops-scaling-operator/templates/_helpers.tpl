{{/* Generate standard labels */}}
{{- define "finops-scaling-operator.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Generate selector labels */}}
{{- define "finops-scaling-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Full name helper */}}
{{- define "finops-scaling-operator.fullname" -}}
{{ .Release.Name }}-{{ .Chart.Name }}
{{- end -}}

{{/* ServiceAccount name */}}
{{- define "finops-scaling-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
  {{- if .Values.serviceAccount.name -}}
    {{ .Values.serviceAccount.name }}
  {{- else -}}
    {{ include "finops-scaling-operator.fullname" . }}
  {{- end -}}
{{- else -}}
  {{ .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}
