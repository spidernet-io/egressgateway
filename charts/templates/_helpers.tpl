{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "project.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name of project .
*/}}
{{- define "project.name" -}}
{{- .Values.global.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
egressgatewayAgent Common labels
*/}}
{{- define "project.egressgatewayAgent.labels" -}}
helm.sh/chart: {{ include "project.chart" . }}
{{ include "project.egressgatewayAgent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
egressgatewayAgent Common labels
*/}}
{{- define "project.egressgatewayController.labels" -}}
helm.sh/chart: {{ include "project.chart" . }}
{{ include "project.egressgatewayController.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
egressgatewayAgent Selector labels
*/}}
{{- define "project.egressgatewayAgent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "project.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Values.agent.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
egressgatewayAgent Selector labels
*/}}
{{- define "project.egressgatewayController.selectorLabels" -}}
app.kubernetes.io/name: {{ include "project.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Values.controller.name | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/* vim: set filetype=mustache: */}}
{{/*
Renders a value that contains template.
Usage:
{{ include "tplvalues.render" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}
{{- define "tplvalues.render" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}




{{/*
Return the appropriate apiVersion for poddisruptionbudget.
*/}}
{{- define "capabilities.policy.apiVersion" -}}
{{- if semverCompare "<1.21-0" .Capabilities.KubeVersion.Version -}}
{{- print "policy/v1beta1" -}}
{{- else -}}
{{- print "policy/v1" -}}
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for deployment.
*/}}
{{- define "capabilities.deployment.apiVersion" -}}
{{- if semverCompare "<1.14-0" .Capabilities.KubeVersion.Version -}}
{{- print "extensions/v1beta1" -}}
{{- else -}}
{{- print "apps/v1" -}}
{{- end -}}
{{- end -}}


{{/*
Return the appropriate apiVersion for RBAC resources.
*/}}
{{- define "capabilities.rbac.apiVersion" -}}
{{- if semverCompare "<1.17-0" .Capabilities.KubeVersion.Version -}}
{{- print "rbac.authorization.k8s.io/v1beta1" -}}
{{- else -}}
{{- print "rbac.authorization.k8s.io/v1" -}}
{{- end -}}
{{- end -}}

{{/*
return the egressgatewayAgent image
*/}}
{{- define "project.egressgatewayAgent.image" -}}
{{- $registryName := .Values.agent.image.registry -}}
{{- $repositoryName := .Values.agent.image.repository -}}
{{- if .Values.global.imageRegistryOverride }}
    {{- printf "%s/%s" .Values.global.imageRegistryOverride $repositoryName -}}
{{ else if $registryName }}
    {{- printf "%s/%s" $registryName $repositoryName -}}
{{- else -}}
    {{- printf "%s" $repositoryName -}}
{{- end -}}
{{- if .Values.agent.image.digest }}
    {{- print "@" .Values.agent.image.digest -}}
{{- else if .Values.global.imageTagOverride -}}
    {{- printf ":%s" .Values.global.imageTagOverride -}}
{{- else if .Values.agent.image.tag -}}
    {{- printf ":%s" .Values.agent.image.tag -}}
{{- else -}}
    {{- printf ":v%s" .Chart.AppVersion -}}
{{- end -}}
{{- end -}}


{{/*
return the egressgatewayController image
*/}}
{{- define "project.egressgatewayController.image" -}}
{{- $registryName := .Values.controller.image.registry -}}
{{- $repositoryName := .Values.controller.image.repository -}}
{{- if .Values.global.imageRegistryOverride }}
    {{- printf "%s/%s" .Values.global.imageRegistryOverride $repositoryName -}}
{{ else if $registryName }}
    {{- printf "%s/%s" $registryName $repositoryName -}}
{{- else -}}
    {{- printf "%s" $repositoryName -}}
{{- end -}}
{{- if .Values.controller.image.digest }}
    {{- print "@" .Values.controller.image.digest -}}
{{- else if .Values.global.imageTagOverride -}}
    {{- printf ":%s" .Values.global.imageTagOverride -}}
{{- else if .Values.controller.image.tag -}}
    {{- printf ":%s" .Values.controller.image.tag -}}
{{- else -}}
    {{- printf ":v%s" .Chart.AppVersion -}}
{{- end -}}
{{- end -}}


{{/*
generate the CA cert
*/}}
{{- define "generate-ca-certs" }}
    {{- $ca := genCA "spidernet.io" (.Values.controller.tls.auto.caExpiration | int) -}}
    {{- $_ := set . "ca" $ca -}}
{{- end }}
