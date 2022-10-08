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
!
{{- range $vrf, $t := .Ports.Vrfs }}
vrf vrf{{ $t.VNI }}
 vni {{ $t.VNI }}
 exit-vrf
!
{{- end }}
line vty
!
