[Match]
Name={{ .Name }}

[Link]
MTUBytes=9216
{{- if .Down }}
ActivationPolicy=always-down
{{- end }}

[Network]
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
LLDP=no
EmitLLDP=no
