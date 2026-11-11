{{- $ASN := .ASN -}}{{- $RouterId := .Loopback -}}! The frr version is not rendered since it seems to be optional.
frr defaults datacenter
hostname {{ .Name }}
username cumulus nopassword
service integrated-vtysh-config
!
log syslog {{ .LogLevel }}
debug bgp updates
debug bgp nht
debug bgp update-groups
debug bgp zebra
{{- range .FRRConfig.VRFs }}
!
vrf {{ . }}
 vni {{ slice . 3 }}
 exit-vrf
{{- end }}
{{- range .FRRConfig.UnnumberedInterfaces }}
!
interface {{ .Name }}{{- if .Vrf }} vrf {{ .Vrf }} {{- end }}
 ipv6 nd ra-interval 6
 no ipv6 nd suppress-ra
{{- end }}
!
{{- range .FRRConfig.BGPs }}
{{- $bgp := . }}
router bgp {{ $ASN }}{{- if .Vrf }} vrf {{ .Vrf }} {{- end }}
 bgp router-id {{ $RouterId }}
 bgp bestpath as-path multipath-relax
 {{- range .PeerGroups }}
 {{- $peergroup := .Name }}
 neighbor {{ $peergroup }} peer-group
 {{- range .Commands }}
 neighbor {{ $peergroup }} {{ . }}
 {{- end }}
 {{- end }}
 {{- range .Neighbors }}
 neighbor {{ .Name }}{{- if .PeerGroup }} peer-group {{ .PeerGroup }}{{- end }}
 {{- end }}
 {{- if .AddressFamilies.IPv4Unicast }}
 !
 {{- $af := .AddressFamilies.IPv4Unicast }}
 address-family ipv4 unicast
  {{- if $bgp.Vrf }}
  redistribute connected
  {{- else }}
  redistribute connected route-map LOCALS
  {{- end }}
  {{- range $af.Neighbors -}}
  {{ $neighbor := .Name }}
  {{- range .Commands }}
  neighbor {{ $neighbor }} {{ . }}
  {{- end -}}
  {{- end }}
 exit-address-family
 {{- end }}
 {{- if .AddressFamilies.L2VPNEVPN }}
 !
 {{- $af := .AddressFamilies.L2VPNEVPN }}
 address-family l2vpn evpn
  {{- range $af.GlobalCommands }}
  {{ . }}
  {{- end -}}
  {{- range $af.Neighbors -}}
  {{ $neighbor := .Name }}
  {{- range .Commands }}
  neighbor {{ $neighbor }} {{ . }}
  {{- end -}}
  {{- end }}
 exit-address-family
 {{- end }}
!
{{- end }}
{{- if .FRRConfig.PrefixLists }}
{{- range .FRRConfig.PrefixLists }}
ip prefix-list {{ .Name }} {{ .Spec }}
{{- end }}
!
{{- end }}
{{- if .FRRConfig.RouteMaps }}
{{- range .FRRConfig.RouteMaps }}
route-map {{ .Name }} {{ .Policy }} {{ .Order }}
{{- range .Entries }}
 {{ . }}
{{- end }}
{{- end }}
!
{{- end }}
route-map LOCALS permit 10
 match interface lo
route-map LOCALS permit 12
 match interface vlan4000
!
line vty
!
