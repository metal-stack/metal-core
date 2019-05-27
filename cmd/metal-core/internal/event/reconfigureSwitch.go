package event

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metal/metal-core/switcher"
	"git.f-i-ts.de/cloud-native/metal/metal-core/vlan"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func buildSwitcherConfig(conf *domain.Config, s *models.MetalSwitch) (*switcher.Conf, error) {
	c := &switcher.Conf{}
	c.Name = s.Name
	asn64, err := strconv.ParseUint(conf.ASN, 10, 32)
	asn := uint32(asn64)
	if err != nil {
		return nil, err
	}
	c.ASN = asn
	c.Loopback = conf.LoopbackIP
	c.MetalCoreCIDR = conf.CIDR
	c.AdditionalBridgeVIDs = conf.AdditionalBridgeVIDs
	c.Neighbors = strings.Split(conf.SpineUplinks, ",")
	c.Tenants = make(map[string]*switcher.Tenant)
	c.Unprovisioned = []string{}
	c.BladePorts = conf.AdditionalBridgePorts
	for _, nic := range s.Nics {
		if contains(conf.AdditionalBridgePorts, *nic.Name) {
			continue
		}
		tenant := &switcher.Tenant{}
		if t, has := c.Tenants[nic.Vrf]; has {
			tenant = t
		}
		if nic.Vrf == "" {
			if !contains(c.Neighbors, *nic.Name) {
				c.Unprovisioned = append(c.Unprovisioned, *nic.Name)
			}
			continue
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(nic.Vrf, "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		tenant.VNI = uint32(vni64)
		tenant.Neighbors = append(tenant.Neighbors, *nic.Name)
		c.Tenants[nic.Vrf] = tenant
	}
	m, err := vlan.ReadMapping()
	if err != nil {
		return nil, err
	}
	err = c.FillVLANIDs(m)
	if err != nil {
		return nil, err
	}
	return c, nil
}

var mux sync.Mutex

func (h *eventHandler) ReconfigureSwitch(switchID string) error {
	mux.Lock()
	defer mux.Unlock()
	params := sw.NewFindSwitchParams()
	params.ID = switchID
	fsr, err := h.SwitchClient.FindSwitch(params)
	if err != nil {
		return errors.Wrap(err, "could not fetch switch from metal-api")
	}

	s := fsr.Payload
	c, err := buildSwitcherConfig(h.Config, s)
	if err != nil {
		return errors.Wrap(err, "could not build switcher config")
	}

	devMode := h.Config.PartitionID == "vagrant-lab"
	err = fillEth0Info(c, h.Config.ManagementGateway, devMode)
	if err != nil {
		return errors.Wrap(err, "could not gather information about eth0 nic")
	}

	zapup.MustRootLogger().Info("Assembled new config for switch",
		zap.Any("config", c))
	if !h.Config.ReconfigureSwitch {
		zapup.MustRootLogger().Info("Skip config application because of environment setting")
		return nil
	}
	err = c.Apply()
	if err != nil {
		return errors.Wrap(err, "could not apply switch config")
	}
	return nil
}

func fillEth0Info(c *switcher.Conf, gw string, devMode bool) error {
	c.Eth0 = switcher.Nic{}
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
	s, _ := addrs[0].IPNet.Mask.Size()
	c.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Eth0.Gateway = gw
	c.DevMode = devMode
	return nil
}

func contains(l []string, e string) bool {
	for _, i := range l {
		if i == e {
			return true
		}
	}
	return false
}
