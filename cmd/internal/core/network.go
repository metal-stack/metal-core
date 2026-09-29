package core

import (
	"fmt"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/net"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	"github.com/vishvananda/netlink"
)

// localNetwork supplies the host observations used alongside the API's desired state.
// Keeping these reads separate lets integration tests run without switch interfaces.
type localNetwork interface {
	fillManagementInfo(*types.Conf, string) error
	linkStatus(string) (apiv2.SwitchPortStatus, error)
	vlanMapping() (vlan.Mapping, error)
}

type systemNetwork struct{}

func (systemNetwork) fillManagementInfo(c *types.Conf, gw string) error {
	c.Ports.Eth0 = types.Nic{}
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}
	addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	if len(addrs) < 1 {
		return fmt.Errorf("there is no ip address configured at eth0")
	}

	ip := addrs[0].IP
	s, _ := addrs[0].Mask.Size()
	c.Ports.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Ports.Eth0.Gateway = gw
	return nil
}

func (systemNetwork) linkStatus(name string) (apiv2.SwitchPortStatus, error) {
	return net.GetLinkStatus(name)
}

func (systemNetwork) vlanMapping() (vlan.Mapping, error) {
	return vlan.ReadMapping()
}
