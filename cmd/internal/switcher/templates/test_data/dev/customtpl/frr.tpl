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
{{- range .FRRConfig.L3VNIs }}
!
vrf Vrf{{ . }}
 vni {{ . }}
 exit-vrf
{{- end }}
{{- range .FRRConfig.UnnumberedInterfaces }}
!
interface {{ .Name }}{{- if .Vrf }} vrf {{ .Vrf }} {{- end }}
 ipv6 nd ra-interval 6
 no ipv6 nd suppress-ra
{{- end }}
!
{{- range .FRRConfig.BGP }}
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
 {{- range .AddressFamilies }}
 {{- if ne .AddressFamily "ipv6 unicast" }}
 !
 address-family {{ .AddressFamily }}
  {{- if eq .AddressFamily "ipv4 unicast" }}
  redistribute connected {{- if not $bgp.Vrf }} route-map LOCALS {{- end }}
  {{- else }}
  {{- range .GlobalStatements }}
  {{ . }}
  {{- end }}
  {{- end -}}
  {{- range .Neighbors -}}
  {{ $neighbor := .Neighbor }}
  {{- range .Commands }}
  neighbor {{ $neighbor }} {{ . }}
  {{- end -}}
  {{- if .RouteMapIn }}
  neighbor {{ .Neighbor }} route-map {{ .RouteMapIn }} in
  {{- end -}}
  {{- if .RouteMapOut }}
  neighbor {{ .Neighbor }} route-map {{ .RouteMapOut }} out
  {{- end }}
  {{- end }}
 exit-address-family
 {{- end }}
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
