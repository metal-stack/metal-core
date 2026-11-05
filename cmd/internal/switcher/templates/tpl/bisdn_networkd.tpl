{{- $IPLoopback := .Loopback -}}
{{- $PXEVlanID := .PXEVlanID -}}
{{- $PXEVni := printf "10%04d" $PXEVlanID -}}
{{- $AdminStatus := .Ports.AdminStatus -}}
{{- $Vrfs := .Ports.Vrfs -}}
{{- $AdditionalBridgeVIDs := .AdditionalBridgeVIDs -}}
#> 10-bisdn-lo.network
[Match]
Name=lo

[Network]
Address={{ $IPLoopback }}/32
{{- range .Ports.Underlay }}

#> 20-bisdn-{{ . }}.network
[Match]
Name={{ . }}

[Link]
MTUBytes=9216
{{- if eq (index $AdminStatus .) "down" }}
ActivationPolicy=always-down
{{- end }}

[Network]
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
LLDP=no
EmitLLDP=no
{{- end }}
{{- range .Ports.Firewalls }}

#> 20-bisdn-{{ .Port }}.network
[Match]
Name={{ .Port }}

[Link]
MTUBytes=9216
{{- if eq (index $AdminStatus .Port) "down" }}
ActivationPolicy=always-down
{{- end }}

[Network]
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
LLDP=no
EmitLLDP=no
{{- end }}
{{- range $vrf, $t := $Vrfs }}

#> 30-bisdn-{{ $vrf }}.netdev
[NetDev]
Name={{ $vrf }}
Kind=vrf

[VRF]
Table={{ $t.VNI }}

#> 30-bisdn-{{ $vrf }}.network
[Match]
Name={{ $vrf }}
{{- end }}

#> 40-bisdn-bridge.netdev
[NetDev]
Name=bridge
Kind=bridge
MTUBytes=9000

[Bridge]
VLANFiltering=yes
STP=no
DefaultPVID=none

#> 40-bisdn-bridge.network
[Match]
Name=bridge

[Network]
VLAN=vlan{{ $PXEVlanID }}
{{- range $vrf, $t := $Vrfs }}
VLAN=vlan{{ $t.VLANID }}
{{- end }}
LinkLocalAddressing=no

[BridgeVLAN]
VLAN={{ $PXEVlanID }}
{{- range $vrf, $t := $Vrfs }}

[BridgeVLAN]
VLAN={{ $t.VLANID }}
{{- end }}
{{- range $AdditionalBridgeVIDs }}

[BridgeVLAN]
VLAN={{ . }}
{{- end }}

#> 41-bisdn-vlan{{ $PXEVlanID }}.netdev
[NetDev]
Name=vlan{{ $PXEVlanID }}
Kind=vlan
MTUBytes=9000

[VLAN]
Id={{ $PXEVlanID }}

#> 41-bisdn-vlan{{ $PXEVlanID }}.network
[Match]
Name=vlan{{ $PXEVlanID }}

[Network]
Address={{ .MetalCoreCIDR }}

#> 42-bisdn-vni{{ $PXEVni }}.netdev
[NetDev]
Name=vni{{ $PXEVni }}
Kind=vxlan
MTUBytes=9000

[VXLAN]
VNI={{ $PXEVni }}
Local={{ $IPLoopback }}
DestinationPort=4789
MacLearning=no
Independent=yes

#> 42-bisdn-vni{{ $PXEVni }}.network
[Match]
Name=vni{{ $PXEVni }}

[Network]
Bridge=bridge
LinkLocalAddressing=no

[Bridge]
Learning=no
NeighborSuppression=no

[BridgeVLAN]
VLAN={{ $PXEVlanID }}
PVID={{ $PXEVlanID }}
EgressUntagged={{ $PXEVlanID }}
{{- range $vrf, $t := $Vrfs }}

#> 41-bisdn-vlan{{ $t.VLANID }}.netdev
[NetDev]
Name=vlan{{ $t.VLANID }}
Kind=vlan
MTUBytes=9000

[VLAN]
Id={{ $t.VLANID }}

#> 41-bisdn-vlan{{ $t.VLANID }}.network
[Match]
Name=vlan{{ $t.VLANID }}

[Network]
VRF={{ $vrf }}

#> 42-bisdn-vni{{ $t.VNI }}.netdev
[NetDev]
Name=vni{{ $t.VNI }}
Kind=vxlan
MTUBytes=9000

[VXLAN]
VNI={{ $t.VNI }}
Local={{ $IPLoopback }}
DestinationPort=4789
MacLearning=no
Independent=yes

#> 42-bisdn-vni{{ $t.VNI }}.network
[Match]
Name=vni{{ $t.VNI }}

[Network]
Bridge=bridge
LinkLocalAddressing=no

[Bridge]
Learning=no
NeighborSuppression=yes

[BridgeVLAN]
VLAN={{ $t.VLANID }}
PVID={{ $t.VLANID }}
EgressUntagged={{ $t.VLANID }}
{{- range $t.Neighbors }}

#> 50-bisdn-{{ . }}.network
[Match]
Name={{ . }}

[Link]
MTUBytes=9000
{{- if eq (index $AdminStatus .) "down" }}
ActivationPolicy=always-down
{{- end }}

[Network]
VRF={{ $vrf }}
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
{{- end }}
{{- end }}
{{- range .Ports.Unprovisioned }}

#> 51-bisdn-{{ . }}.network
[Match]
Name={{ . }}

[Link]
MTUBytes=9000
{{- if eq (index $AdminStatus .) "down" }}
ActivationPolicy=always-down
{{- end }}

[Network]
Bridge=bridge
LinkLocalAddressing=no
LLDP=no
EmitLLDP=no

[BridgeVLAN]
VLAN={{ $PXEVlanID }}
PVID={{ $PXEVlanID }}
EgressUntagged={{ $PXEVlanID }}
{{- end }}
