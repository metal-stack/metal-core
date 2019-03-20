package event

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/switcher"
	"git.f-i-ts.de/cloud-native/metallib/vlan"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
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
	c.Neighbors = strings.Split(conf.SpineUplinks, ",")

	c.Eth0 = switcher.Nic{}
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return nil, err
	}
	addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	if len(addrs) < 1 {
		return nil, fmt.Errorf("there is no ip address configured at eth0")
	}

	eth0Addr := addrs[0]
	permanent := unix.IFA_F_PERMANENT & eth0Addr.Flags
	zapup.MustRootLogger().Debug("eth0", zap.Int("flags", eth0Addr.Flags))
	if permanent != 0 {
		zapup.MustRootLogger().Info("eth0 ip address was assigned statically, reuse this setting")
		ip := eth0Addr.IP
		n := eth0Addr.IPNet
		masked := ip.Mask(n.Mask)
		gw := net.IPv4(masked[0], masked[1], masked[2], masked[3]+1)
		c.Eth0.Dhcp = false
		c.Eth0.AddressCIDR = ip.String()
		c.Eth0.Gateway = gw.String()
	} else {
		zapup.MustRootLogger().Info("eth0 ip address was assigned with dhcp, reuse this setting")
		c.Eth0.Dhcp = true
	}

	c.Tenants = make(map[string]*switcher.Tenant)
	c.Unprovisioned = []string{}
	for _, nic := range s.Nics {
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

func (h *eventHandler) ReconfigureSwitch(switchID string) {
	mux.Lock()
	defer mux.Unlock()
	params := sw.NewFindSwitchParams()
	params.ID = switchID
	fsr, err := h.SwitchClient.FindSwitch(params)
	if err != nil {
		zapup.MustRootLogger().Error("Could not fetch switch from metal-api",
			zap.Error(err))
		return
	}

	s := fsr.Payload
	c, err := buildSwitcherConfig(h.Config, s)
	if err != nil {
		zapup.MustRootLogger().Error("Could not build switcher config",
			zap.Error(err))
		return
	}

	zapup.MustRootLogger().Info("Would apply this configuration to the switch",
		zap.Any("config", c))
	if !h.Config.ReconfigureSwitch {
		zapup.MustRootLogger().Info("Skip configuration application because of environment setting")
		return
	}
	err = c.Apply()
	if err != nil {
		zapup.MustRootLogger().Error("Could not apply switch config",
			zap.Error(err))
	}
}

func contains(l []string, e string) bool {
	for _, i := range l {
		if i == e {
			return true
		}
	}
	return false
}
