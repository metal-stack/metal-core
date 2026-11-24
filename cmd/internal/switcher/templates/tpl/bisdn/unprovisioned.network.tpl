[Match]
Name={{ .Name }}

[Link]
MTUBytes=9000
{{- if .Down }}
ActivationPolicy=always-down
{{- end }}

[Network]
Bridge=bridge
LinkLocalAddressing=no
LLDP=no
EmitLLDP=no

[BridgeVLAN]
VLAN={{ .VLANID }}
PVID={{ .VLANID }}
EgressUntagged={{ .VLANID }}
