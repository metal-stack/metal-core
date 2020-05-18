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
    address {{ .Ports.Eth0.AddressCIDR }}
    gateway {{ .Ports.Eth0.Gateway }}
    vrf mgmt
{{- if .DevMode  }}

auto vagrant
iface vagrant inet dhcp
{{- end }}

auto mgmt
iface mgmt
    address 127.0.0.1/8
    vrf-table auto
{{- range .Ports.Underlay }}

auto {{ . }}
iface {{ . }}
    mtu 9216
{{- end }}
{{- range .Ports.Firewalls }}

auto {{ .Port }}
iface {{ .Port }}
    mtu 9216
{{- end }}

auto bridge
iface bridge
    bridge-ports vni104000{{ range .Ports.Unprovisioned }} {{ . }}{{ end }}{{ range .Ports.BladePorts }} {{ . }}{{ end }}{{ range $vrf, $t := .Ports.Vrfs }} vni{{ $t.VNI }}{{ end }}
    bridge-vids 4000{{ range $vrf, $t := .Ports.Vrfs }} {{ $t.VLANID }}{{ end }}{{ range $vids := .AdditionalBridgeVIDs }} {{ $vids }}{{ end }}
    bridge-vlan-aware yes

# Tenants
{{- range $vrf, $t := .Ports.Vrfs }}

auto {{ $vrf }}
iface {{ $vrf }}
    vrf-table auto

{{- range $t.Neighbors }}

auto {{ . }}
iface {{ . }}
    mtu 9000
    post-up sysctl -w net.ipv6.conf.{{ . }}.disable_ipv6=0
    vrf {{ $vrf }}
{{- end }}

auto vlan{{ $t.VLANID }}
iface vlan{{ $t.VLANID }}
    mtu 9000
    vlan-id {{ $t.VLANID }}
    vlan-raw-device bridge
    vrf {{ $vrf }}

auto vni{{ $t.VNI }}
iface vni{{ $t.VNI }}
    mtu 9000
    bridge-access {{ $t.VLANID }}
    bridge-arp-nd-suppress on
    bridge-learning off
    mstpctl-bpduguard yes
    mstpctl-portbpdufilter yes
    vxlan-id {{ $t.VNI }}
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

{{- range .Ports.Unprovisioned }}

auto {{ . }}
iface {{ . }}
    mtu 9000
    bridge-access 4000
{{- end }}