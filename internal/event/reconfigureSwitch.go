package event

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/switcher"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/vlan"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func buildSwitcherConfig(conf *domain.Config, s *models.V1SwitchResponse) (*switcher.Conf, error) {
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
	p := switcher.Ports{
		Underlay:      strings.Split(conf.SpineUplinks, ","),
		Unprovisioned: []string{},
		Vrfs:          map[string]*switcher.Vrf{},
		Firewalls:     map[string]*switcher.Firewall{},
	}
	p.BladePorts = conf.AdditionalBridgePorts
	for _, nic := range s.Nics {
		port := *nic.Name
		if contains(p.Underlay, port) {
			continue
		}
		if contains(conf.AdditionalBridgePorts, port) {
			continue
		}
		if nic.Vrf == "" {
			if !contains(p.Unprovisioned, port) {
				p.Unprovisioned = append(p.Unprovisioned, port)
			}
			continue
		}
		// Firewall-Port
		if nic.Vrf == "default" {
			fw := &switcher.Firewall{
				Port: port,
			}
			if nic.Filter != nil {
				fw.Vnis = nic.Filter.Vnis
				fw.Cidrs = nic.Filter.Cidrs
			}
			p.Firewalls[port] = fw
			continue
		}
		// Machine-Port
		vrf := &switcher.Vrf{}
		if v, has := p.Vrfs[nic.Vrf]; has {
			vrf = v
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(nic.Vrf, "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		vrf.VNI = uint32(vni64)
		vrf.Neighbors = append(vrf.Neighbors, port)
		if nic.Filter != nil {
			vrf.Cidrs = nic.Filter.Cidrs
		}
		p.Vrfs[nic.Vrf] = vrf
	}
	c.Ports = p
	c.FillRouteMapsAndIPPrefixLists()
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
	fsr, err := h.SwitchClient.FindSwitch(params, h.Auth)
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
		zapup.MustRootLogger().Debug("Skip config application because of environment setting")
		return nil
	}
	err = c.Apply()
	if err != nil {
		return errors.Wrap(err, "could not apply switch config")
	}
	return nil
}

func fillEth0Info(c *switcher.Conf, gw string, devMode bool) error {
	c.Ports.Eth0 = switcher.Nic{}
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
	c.Ports.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Ports.Eth0.Gateway = gw
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
