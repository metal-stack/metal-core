{{- $IPLoopback := .Loopback -}}
{{- $PXEVlanID := .PXEVlanID -}}
{{- $PXEVni := printf "10%04d" $PXEVlanID -}}
# This file describes the network interfaces available on your system
# and how to activate them. For more information, see interfaces(5).

source /etc/network/interfaces.d/*.intf

# The loopback network interface
auto lo
iface lo inet loopback
    address {{ $IPLoopback }}/32

# The primary network interface
auto eth0
iface eth0
    address {{ .Eth0.AddressCIDR }}
    gateway {{ .Eth0.Gateway }}
    vrf mgmt

auto mgmt
iface mgmt
    address 127.0.0.1/8
    vrf-table auto

{{- range .Ports }}

auto {{ .Name }}
iface {{ .Name }}
    mtu {{ .MTU }}
    {{- if .VRF }}
    post-up sysctl -w net.ipv6.conf.{{ .Name }}.disable_ipv6=0
    vrf {{ .VRF }}
    {{- end }}
    {{- if .VLAN }}
    bridge-access {{ $PXEVlanID }}
    {{- end }}
{{- end }}

auto bridge
iface bridge
    bridge-ports vni{{ $PXEVni }}{{ range .Ports }}{{ if .VLAN }} {{ .Name }}{{ end }}{{ end }}{{ range .Vrfs }} vni{{ .VNI }}{{ end }}
    bridge-vids {{ $PXEVlanID }}{{ range .Vrfs }} {{ .VLANID }}{{ end }}
    bridge-vlan-aware yes

{{- range .Vrfs }}

auto {{ .Name }}
iface {{ .Name }}
    vrf-table auto

auto vlan{{ .VLANID }}
iface vlan{{ .VLANID }}
    mtu 9000
    vlan-id {{ .VLANID }}
    vlan-raw-device bridge
    vrf {{ .Name }}

auto vni{{ .VNI }}
iface vni{{ .VNI }}
    mtu 9000
    bridge-access {{ .VLANID }}
    bridge-arp-nd-suppress on
    bridge-learning off
    mstpctl-bpduguard yes
    mstpctl-portbpdufilter yes
    vxlan-id {{ .VNI }}
    vxlan-local-tunnelip {{ $IPLoopback }}
{{- end }}

# PXE-Config
auto vlan{{ $PXEVlanID }}
iface vlan{{ $PXEVlanID }}
    mtu 9000
    address {{ .MetalCoreCIDR }}
    vlan-id {{ $PXEVlanID }}
    vlan-raw-device bridge

auto vni{{ $PXEVni }}
iface vni{{ $PXEVni }}
    mtu 9000
    bridge-access {{ $PXEVlanID }}
    bridge-learning off
    mstpctl-bpduguard yes
    mstpctl-portbpdufilter yes
    vxlan-id {{ $PXEVni }}
    vxlan-local-tunnelip {{ $IPLoopback }}
