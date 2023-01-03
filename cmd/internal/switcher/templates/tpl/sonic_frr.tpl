{{- $ASN := .ASN -}}{{- $RouterId := .Loopback -}}! The frr version is not rendered since it seems to be optional.
frr defaults datacenter
hostname {{ .Name }}
password zebra
enable password zebra
!
log syslog {{ .LogLevel }}
log facility local4
debug bgp updates
debug bgp nht
debug bgp update-groups
debug bgp zebra
{{- range $vrf, $t := .Ports.Vrfs }}
!
vrf Vrf{{ $t.VNI }}
 vni {{ $t.VNI }}
 exit-vrf
{{- end }}
!
router bgp {{ $ASN }}
 bgp router-id {{ $RouterId }}
 bgp bestpath as-path multipath-relax
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 neighbor FABRIC timers 2 8
 {{- range .Ports.Underlay }}
 neighbor {{ . }} interface peer-group FABRIC
 {{- end }}
 neighbor FIREWALL peer-group
 neighbor FIREWALL remote-as external
 neighbor FIREWALL timers 2 8
 {{- range .Ports.Firewalls }}
 neighbor {{ .Port }} interface peer-group FIREWALL
 {{- end }}
 !
 address-family ipv4 unicast
  redistribute connected route-map LOOPBACKS
  neighbor FIREWALL allowas-in 2
  {{- range $k, $f := .Ports.Firewalls }}
  neighbor {{ $f.Port }} route-map fw-{{ $k }}-in in
  {{- end }}
 exit-address-family
 !
 address-family l2vpn evpn
  advertise-all-vni
  neighbor FABRIC activate
  neighbor FABRIC allowas-in 2
  neighbor FIREWALL activate
  neighbor FIREWALL allowas-in 2
  {{- range $k, $f := .Ports.Firewalls }}
  neighbor {{ $f.Port }} route-map fw-{{ $k }}-vni out
  {{- end }}
 exit-address-family
!
route-map LOOPBACKS permit 10
 match interface Loopback0
!
{{- range $k, $f := .Ports.Firewalls }}
# route-maps for firewall@{{ $k }}
        {{- range $f.IPPrefixLists }}
ip prefix-list {{ .Name }} {{ .Spec }}
        {{- end}}
        {{- range $f.RouteMaps }}
route-map {{ .Name }} {{ .Policy }} {{ .Order }}
                {{- range .Entries }}
 {{ . }}
                {{- end }}
        {{- end }}
!
{{- end }}
{{- if .Ports.Eth0.Gateway }}
ip route 0.0.0.0/0 {{ .Ports.Eth0.Gateway }} nexthop-vrf mgmt
{{- end }}
!
{{- range $vrf, $t := .Ports.Vrfs }}
router bgp {{ $ASN }} vrf {{ $vrf }}
 bgp router-id {{ $RouterId }}
 bgp bestpath as-path multipath-relax
 neighbor MACHINE peer-group
 neighbor MACHINE remote-as external
 neighbor MACHINE timers 2 8
 {{- range $t.Neighbors }}
 neighbor {{ . }} interface peer-group MACHINE
 {{- end }}
 !
 address-family ipv4 unicast
  redistribute connected
  neighbor MACHINE maximum-prefix 24000
  {{- if gt (len $t.IPPrefixLists) 0 }}
  neighbor MACHINE route-map {{ $vrf }}-in in
  {{- end }}
 exit-address-family
 !
 address-family l2vpn evpn
  advertise ipv4 unicast
 exit-address-family
!
{{- if gt (len $t.IPPrefixLists) 0 }}
# route-maps for {{ $vrf }}
        {{- range $t.IPPrefixLists }}
ip prefix-list {{ .Name }} {{ .Spec }}
        {{- end}}
        {{- range $t.RouteMaps }}
route-map {{ .Name }} {{ .Policy }} {{ .Order }}
                {{- range .Entries }}
 {{ . }}
                {{- end }}
        {{- end }}
!{{- end }}{{- end }}
line vty
!