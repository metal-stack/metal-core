package cmd

import "time"

type Config struct {
	// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
	CIDR                      string        `required:"true" desc:"set the metal core CIDR"`
	PartitionID               string        `required:"true" desc:"set the partition ID" envconfig:"partition_id"`
	RackID                    string        `required:"true" desc:"set the rack ID" envconfig:"rack_id"`
	BindAddress               string        `required:"false" default:"0.0.0.0" desc:"set server bind address" split_words:"true"`
	MetricsServerPort         int           `required:"false" default:"2112" desc:"the port of the metrics server" split_words:"true"`
	MetricsServerBindAddress  string        `required:"false" default:"0.0.0.0" desc:"the bind addr of the metrics server" split_words:"true"`
	LogLevel                  string        `required:"false" default:"info" desc:"set log level" split_words:"true"`
	ApiProtocol               string        `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
	ApiIP                     string        `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_ip"`
	ApiPort                   int           `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
	ApiBasePath               string        `required:"false" default:"" desc:"set metal api basepath" envconfig:"metal_api_basepath"`
	LoopbackIP                string        `required:"false" default:"10.0.0.11" desc:"set the loopback ip address that is used with BGP unnumbered" split_words:"true"`
	ASN                       string        `required:"false" default:"420000011" desc:"set the ASN that is used with BGP"`
	SpineUplinks              []string      `required:"false" default:"swp31,swp32" desc:"set the ports that are connected to spines" split_words:"true"`
	ManagementGateway         string        `required:"false" default:"192.168.0.1" desc:"the default gateway for the management network" split_words:"true"`
	ReconfigureSwitch         bool          `required:"false" default:"false" desc:"let metal-core reconfigure the switch" split_words:"true"`
	ReconfigureSwitchInterval time.Duration `required:"false" default:"10s" desc:"pull interval to fetch and apply switch configuration" split_words:"true"`
	AdditionalBridgeVIDs      []string      `required:"false" desc:"additional vlan ids that should be configured at the vlan-aware bridge" envconfig:"additional_bridge_vids"`
	AdditionalBridgePorts     []string      `required:"false" desc:"additional switch ports that should be configured at the vlan-aware bridge" envconfig:"additional_bridge_ports"`
	InterfacesTplFile         string        `required:"false" default:"" desc:"the golang template file used to render /etc/network/interfaces, a default template is included" envconfig:"interfaces_tpl_file"`
	FrrTplFile                string        `required:"false" default:"" desc:"the golang template file used to render /etc/frr/frr.conf, a default template is included" envconfig:"frr_tpl_file"`
	HMACKey                   string        `required:"true" desc:"the preshared key for the hmac calculation" envconfig:"hmac_key"`
	GrpcAddress               string        `required:"true" default:"" desc:"the gRPC address" envconfig:"grpc_address"`
	GrpcCACertFile            string        `required:"false" desc:"the gRPC CA certificate file" envconfig:"grpc_ca_cert_file"`
	GrpcClientCertFile        string        `required:"false" desc:"the gRPC client certificate file" envconfig:"grpc_client_cert_file"`
	GrpcClientKeyFile         string        `required:"false" desc:"the gRPC client key file" envconfig:"grpc_client_key_file"`
	DHCPServers               []string      `required:"false" desc:"DHCP helper address config for SONiC" envconfig:"dhcp_servers"`
}
