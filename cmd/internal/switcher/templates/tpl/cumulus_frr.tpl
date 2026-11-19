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
{{- range .FRRConfig.BGP }}
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
  {{- range $af.GlobalStatements }}
  {{ . }}
  {{- end -}}
  {{- range $af.Neighbors -}}
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
 {{- if .AddressFamilies.IPv6Unicast }}
 !
 {{- $af := .AddressFamilies.IPv6Unicast }}
 address-family ipv6 unicast
  {{- range $af.GlobalStatements }}
  {{ . }}
  {{- end -}}
  {{- range $af.Neighbors -}}
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
 {{- if .AddressFamilies.L2VPNEVPN }}
 !
 {{- $af := .AddressFamilies.L2VPNEVPN }}
 address-family l2vpn evpn
  {{- range $af.GlobalStatements }}
  {{ . }}
  {{- end -}}
  {{- range $af.Neighbors -}}
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
{{- if .Eth0.Gateway }}
{{- $gateway := .Eth0.Gateway }}
ip route 0.0.0.0/0 {{ $gateway }} nexthop-vrf mgmt
{{- range .AdditionalMgmtRoutes }}
ip route {{ . }} {{ $gateway }} nexthop-vrf mgmt
{{- end }}
{{- end }}
!
line vty
!
