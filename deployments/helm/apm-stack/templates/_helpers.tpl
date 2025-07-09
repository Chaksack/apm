{{/*
Expand the name of the chart.
*/}}
{{- define "apm-stack.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "apm-stack.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "apm-stack.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "apm-stack.labels" -}}
helm.sh/chart: {{ include "apm-stack.chart" . }}
{{ include "apm-stack.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.global.labels }}
{{- toYaml .Values.global.labels | nindent 0 }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "apm-stack.selectorLabels" -}}
app.kubernetes.io/name: {{ include "apm-stack.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "apm-stack.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "apm-stack.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the appropriate apiVersion for RBAC resources
*/}}
{{- define "apm-stack.rbac.apiVersion" -}}
{{- if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" -}}
rbac.authorization.k8s.io/v1
{{- else -}}
rbac.authorization.k8s.io/v1beta1
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for Ingress
*/}}
{{- define "apm-stack.ingress.apiVersion" -}}
{{- if .Capabilities.APIVersions.Has "networking.k8s.io/v1" -}}
networking.k8s.io/v1
{{- else if .Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" -}}
networking.k8s.io/v1beta1
{{- else -}}
extensions/v1beta1
{{- end -}}
{{- end -}}

{{/*
Component specific helpers
*/}}

{{/*
Prometheus fullname
*/}}
{{- define "apm-stack.prometheus.fullname" -}}
{{- printf "%s-prometheus" (include "apm-stack.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Grafana fullname
*/}}
{{- define "apm-stack.grafana.fullname" -}}
{{- printf "%s-grafana" (include "apm-stack.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Loki fullname
*/}}
{{- define "apm-stack.loki.fullname" -}}
{{- printf "%s-loki" (include "apm-stack.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Jaeger fullname
*/}}
{{- define "apm-stack.jaeger.fullname" -}}
{{- printf "%s-jaeger" (include "apm-stack.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Alertmanager fullname
*/}}
{{- define "apm-stack.alertmanager.fullname" -}}
{{- printf "%s-alertmanager" (include "apm-stack.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the namespace to use
*/}}
{{- define "apm-stack.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace -}}
{{- end -}}

{{/*
Return the appropriate storage class
*/}}
{{- define "apm-stack.storageClass" -}}
{{- if .Values.global.storageClass -}}
{{- if (eq "-" .Values.global.storageClass) -}}
storageClassName: ""
{{- else }}
storageClassName: {{ .Values.global.storageClass | quote }}
{{- end -}}
{{- end -}}
{{- end -}}