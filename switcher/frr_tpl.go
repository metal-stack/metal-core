package switcher

const frrTPL = `{{- $ASN := .ASN -}}{{- $RouterId := .Loopback -}}! The frr version is not rendered since it seems to be optional.
frr defaults datacenter
hostname {{ .Name }}
username cumulus nopassword
service integrated-vtysh-config
log syslog informational
{{- range $vrf, $t := .Tenants }}
!
vrf vrf{{ $t.VNI }}
 vni {{ $t.VNI }}
{{- end }}
{{- range .Neighbors }}
!
interface {{ . }}
 ipv6 nd ra-interval 6
 no ipv6 nd suppress-ra
{{- end }}
{{- range $vrf, $t := .Tenants }}
{{- range $t.Neighbors }}
!
interface {{ . }}
 ipv6 nd ra-interval 6
 no ipv6 nd suppress-ra
{{- end }}
{{- end }}
!
router bgp {{ $ASN }}
 bgp router-id {{ $RouterId }}
 bgp bestpath as-path multipath-relax
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 neighbor FABRIC timers 1 3
 {{- range .Neighbors }}
 neighbor {{ . }} interface peer-group FABRIC
 {{- end }}
 !
 address-family ipv4 unicast
  redistribute connected route-map LOOPBACKS
 exit-address-family
 !
 address-family l2vpn evpn
  neighbor FABRIC activate
  advertise-all-vni
 exit-address-family
!
route-map LOOPBACKS permit 10
 match interface lo
!
ip route 0.0.0.0/0 {{ .Eth0.Gateway }} nexthop-vrf mgmt
!
{{- range $vrf, $t := .Tenants }}
router bgp {{ $ASN }} vrf {{ $vrf }}
 bgp router-id {{ $RouterId }}
 bgp bestpath as-path multipath-relax
 neighbor MACHINE peer-group
 neighbor MACHINE remote-as external
 neighbor MACHINE timers 1 3
 {{- range $t.Neighbors }}
 neighbor {{ . }} interface peer-group MACHINE
 {{- end }}
 !
 address-family ipv4 unicast
  redistribute connected
  neighbor MACHINE maximum-prefix 100
 exit-address-family
 !
 address-family l2vpn evpn
  advertise ipv4 unicast
 exit-address-family
!{{- end }}
line vty
!`
