package switcher

// Conf holds the switch configuration
type Conf struct {
	Name                 string
	Loopback             string
	ASN                  uint32
	Eth0                 Nic
	Neighbors            []string
	Tenants              map[string]*Tenant
	Unprovisioned        []string
	DevMode              bool
	MetalCoreCIDR        string
	AdditionalBridgeVIDs []string
	BladePorts           []string
}

// Tenant holds the switch configuration for a specific tenant
type Tenant struct {
	VNI       uint32
	VLANID    uint16
	Neighbors []string
}

// Nic holds the configuration for a network interface
type Nic struct {
	AddressCIDR string
	Gateway     string
}
