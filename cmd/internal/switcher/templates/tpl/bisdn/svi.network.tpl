[Match]
Name={{ .Name }}

[Network]
{{- if .VRF }}
VRF={{ .VRF }}
{{- end }}
{{- if .Address }}
Address={{ .Address }}
{{- end }}
