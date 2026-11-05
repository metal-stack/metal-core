[Match]
Name={{ .Name }}

[Network]
Bridge=bridge
LinkLocalAddressing=no

[Bridge]
Learning=no
NeighborSuppression={{ if .NeighborSuppression }}yes{{ else }}no{{ end }}

[BridgeVLAN]
VLAN={{ .VLANID }}
PVID={{ .VLANID }}
EgressUntagged={{ .VLANID }}
