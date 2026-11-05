[Match]
Name={{ .Name }}

[Link]
MTUBytes=9000
{{- if .Down }}
ActivationPolicy=always-down
{{- end }}

[Network]
VRF={{ .VRF }}
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
