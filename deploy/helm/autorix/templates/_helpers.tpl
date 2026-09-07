{{/*
Expand the name of the chart.
*/}}
{{- define "autorix.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "autorix.fullname" -}}
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
Common labels
*/}}
{{- define "autorix.labels" -}}
helm.sh/chart: {{ include "autorix.fullname" . }}
app.kubernetes.io/name: {{ include "autorix.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Baseline pod security context applied to every Autorix workload.
*/}}
{{- define "autorix.podSecurityContext" -}}
{{- toYaml .Values.global.securityContext.pod -}}
{{- end -}}

{{/*
Baseline container security context with an optional workload-specific override.
*/}}
{{- define "autorix.containerSecurityContext" -}}
{{- $defaults := deepCopy .root.Values.global.securityContext.container -}}
{{- $overrides := default (dict) .override -}}
{{- toYaml (mergeOverwrite $defaults $overrides) -}}
{{- end -}}

{{/*
Availability defaults are intentionally soft: hostname is universally present
on Kubernetes nodes, while ScheduleAnyway and preferred anti-affinity keep
small or heterogeneous self-hosted clusters schedulable during an outage.
*/}}
{{- define "autorix.workloadAvailability" -}}
{{- $availability := .root.Values.global.availability -}}
terminationGracePeriodSeconds: {{ $availability.terminationGracePeriodSeconds }}
{{- if $availability.topologySpread.enabled }}
topologySpreadConstraints:
  - maxSkew: {{ $availability.topologySpread.maxSkew }}
    topologyKey: {{ $availability.topologySpread.topologyKey | quote }}
    whenUnsatisfiable: {{ $availability.topologySpread.whenUnsatisfiable }}
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "autorix.name" .root }}
        app.kubernetes.io/component: {{ .component }}
{{- end }}
{{- if $availability.podAntiAffinity.enabled }}
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: {{ $availability.podAntiAffinity.weight }}
        podAffinityTerm:
          topologyKey: {{ $availability.podAntiAffinity.topologyKey | quote }}
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: {{ include "autorix.name" .root }}
              app.kubernetes.io/component: {{ .component }}
{{- end }}
{{- end -}}

{{/*
Optional service-mesh injection annotations. Provider validation prevents a
misspelled profile from silently disabling transport protection.
*/}}
{{- define "autorix.transportSecurityAnnotations" -}}
{{- $provider := default "none" .Values.global.transportSecurity.provider -}}
{{- if eq $provider "istio" }}
sidecar.istio.io/inject: "true"
{{- else if eq $provider "linkerd" }}
linkerd.io/inject: "enabled"
{{- else if eq $provider "spire" }}
{{/* SPIRE workload registration and SVID delivery are operator-owned. Do not
     infer annotations or CRDs from this chart because trust domains and node
     attestation are deployment-specific. */}}
{{- else if ne $provider "none" }}
{{- fail (printf "global.transportSecurity.provider must be one of none, istio, linkerd, or spire; got %q" $provider) }}
{{- end }}
{{- end -}}


{{/*
Render an immutable image reference. Production chart defaults deliberately
fail closed until a release process supplies a verified OCI digest.
*/}}
{{- define "autorix.image" -}}
{{- $image := .image -}}
{{- $repository := required "image.repository is required" $image.repository -}}
{{- $digest := default "" $image.digest -}}
{{- if and .root.Values.global.imageDigests.required (not (regexMatch "^sha256:[a-f0-9]{64}$" $digest)) -}}
{{- fail (printf "image digest is required and must be sha256:<64 lowercase hex chars> for %s" $repository) -}}
{{- end -}}
{{- if $digest -}}
{{- printf "%s@%s" $repository $digest -}}
{{- else -}}
{{- printf "%s:%s" $repository (required "image.tag is required when image digest enforcement is disabled" $image.tag) -}}
{{- end -}}
{{- end -}}
