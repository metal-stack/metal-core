[NetDev]
Name={{ .Name }}
Kind=vxlan
MTUBytes=9000

[VXLAN]
VNI={{ .VNI }}
Local={{ .Loopback }}
DestinationPort=4789
MacLearning=no
Independent=yes
