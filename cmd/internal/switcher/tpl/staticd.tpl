frr defaults datacenter
hostname {{ .Name }}
password zebra
enable password zebra
!
log syslog {{ .LogLevel }}
log facility local4
!
{{- if .Ports.Eth0.Gateway }}
ip route 0.0.0.0/0 {{ .Ports.Eth0.Gateway }} nexthop-vrf mgmt
{{- end }}
!
line vty
!
