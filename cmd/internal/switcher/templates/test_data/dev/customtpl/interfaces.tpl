{{- $IPLoopback := .Loopback -}}
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
    bridge-access 4000
    {{- end }}
{{- end }}

auto bridge
iface bridge
    bridge-ports vni104000{{ range .Ports }}{{ if .VLAN }} {{ .Name }}{{ end }}{{ end }}{{ range .Vrfs }} vni{{ .VNI }}{{ end }}
    bridge-vids 4000{{ range .Vrfs }} {{ .VLANID }}{{ end }}
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
auto vlan4000
iface vlan4000
    mtu 9000
    address {{ .MetalCoreCIDR }}
    vlan-id 4000
    vlan-raw-device bridge

auto vni104000
iface vni104000
    mtu 9000
    bridge-access 4000
    bridge-learning off
    mstpctl-bpduguard yes
    mstpctl-portbpdufilter yes
    vxlan-id 104000
    vxlan-local-tunnelip {{ $IPLoopback }}
