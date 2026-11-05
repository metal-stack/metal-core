[Match]
Name={{ .Name }}

[Network]
{{- range .SVIs }}
VLAN={{ . }}
{{- end }}
LinkLocalAddressing=no
{{- range .VLANs }}

[BridgeVLAN]
VLAN={{ . }}
{{- end }}
