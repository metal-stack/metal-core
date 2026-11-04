{{- $ASN := .ASN -}}{{- $RouterId := .Loopback -}}! The frr version is not rendered since it seems to be optional.
frr defaults datacenter
hostname {{ .Name }}
password zebra
enable password zebra
!
agentx
log syslog {{ .LogLevel }}
log facility local4
debug bgp updates
debug bgp nht
debug bgp update-groups
debug bgp zebra
debug zebra events
debug zebra nexthop detail
debug zebra rib detailed
debug zebra nht detailed
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
 !
 address-family {{ .AddressFamily }}
  {{- range .GlobalStatements }}
  {{ . }}
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
route-map DENY_MGMT deny 10
  match interface eth0
route-map DENY_MGMT permit 20
{{- if .SetSrcLoopback }}
!
route-map RM_SET_SRC permit 10
 set src {{ .Loopback }}
exit
!
ip protocol bgp route-map RM_SET_SRC
{{- end }}
!
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
