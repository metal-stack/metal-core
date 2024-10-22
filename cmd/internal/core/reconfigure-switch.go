package core

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	sw "github.com/metal-stack/metal-go/api/client/switch_operations"
	"github.com/metal-stack/metal-go/api/models"
)

// ReconfigureSwitch reconfigures the switch.
func (c *Core) ReconfigureSwitch() {
	t := time.NewTicker(c.reconfigureSwitchInterval)
	host, _ := os.Hostname()
	for range t.C {
		c.log.Info("trigger reconfiguration")
		start := time.Now()
		s, err := c.reconfigureSwitch(host)
		elapsed := time.Since(start)
		c.log.Info("reconfiguration took", "elapsed", elapsed)

		params := sw.NewNotifySwitchParams()
		params.ID = host
		ns := elapsed.Nanoseconds()
		nr := &models.V1SwitchNotifyRequest{
			SyncDuration: &ns,
			PortStates:   make(map[string]string),
		}
		if err != nil {
			errStr := err.Error()
			nr.Error = &errStr
			c.log.Error("reconfiguration failed", "error", err)
			c.metrics.CountError("switch-reconfiguration")
		} else {
			c.log.Info("reconfiguration succeeded")
		}

		// fill the port states of the switch
		var nics []*models.V1SwitchNic
		if s != nil {
			nics = s.Nics
		}
		for _, n := range nics {
			if n == nil || n.Name == nil {
				// lets log the whole nic because the name could be empty; lets hope there is some useful information
				// in the nic
				c.log.Error("could not check if link is up", "nic", n)
				c.metrics.CountError("switch-reconfiguration")
				continue
			}
			isup, err := isLinkUp(*n.Name)
			if err != nil {
				c.log.Error("could not check if link is up", "error", err, "nicname", *n.Name)
				nr.PortStates[*n.Name] = models.V1SwitchNicActualUNKNOWN
				c.metrics.CountError("switch-reconfiguration")
				continue
			}
			if isup {
				nr.PortStates[*n.Name] = models.V1SwitchNicActualUP
			} else {
				nr.PortStates[*n.Name] = models.V1SwitchNicActualDOWN
			}

		}

		params.Body = nr
		_, err = c.driver.SwitchOperations().NotifySwitch(params, nil)
		if err != nil {
			c.log.Error("notification about switch reconfiguration failed", "error", err)
			c.metrics.CountError("reconfiguration-notification")
		}
	}
}

func (c *Core) reconfigureSwitch(switchName string) (*models.V1SwitchResponse, error) {
	params := sw.NewFindSwitchParams()
	params.ID = switchName
	fsr, err := c.driver.SwitchOperations().FindSwitch(params, nil)
	if err != nil {
		return nil, fmt.Errorf("could not fetch switch from metal-api: %w", err)
	}

	s := fsr.Payload
	switchConfig, err := c.buildSwitcherConfig(s)
	if err != nil {
		return nil, fmt.Errorf("could not build switcher config: %w", err)
	}

	err = fillEth0Info(switchConfig, c.managementGateway)
	if err != nil {
		return nil, fmt.Errorf("could not gather information about eth0 nic: %w", err)
	}

	c.log.Debug("assembled new config for switch", "config", switchConfig)
	if !c.enableReconfigureSwitch {
		c.log.Debug("skip config application because of environment setting")
		return s, nil
	}

	err = c.nos.Apply(switchConfig)
	if err != nil {
		return nil, fmt.Errorf("could not apply switch config: %w", err)
	}

	return s, nil
}

func (c *Core) buildSwitcherConfig(s *models.V1SwitchResponse) (*types.Conf, error) {
	asn64, err := strconv.ParseUint(c.asn, 10, 32)
	if err != nil {
		return nil, err
	}
	if c.pxeVlanID >= vlan.VlanIDMin && c.pxeVlanID <= vlan.VlanIDMax {
		return nil, fmt.Errorf("configured PXE VLAN ID is in the reserved area of %d, %d", vlan.VlanIDMin, vlan.VlanIDMax)
	}

	switcherConfig := &types.Conf{
		Name:                    s.Name,
		LogLevel:                mapLogLevel(c.logLevel),
		ASN:                     uint32(asn64), // nolint:gosec
		Loopback:                c.loopbackIP,
		MetalCoreCIDR:           c.cidr,
		AdditionalBridgeVIDs:    c.additionalBridgeVIDs,
		PXEVlanID:               c.pxeVlanID,
		AdditionalRouteMapCIDRs: c.additionalRouteMapCIDRs,
	}

	p := types.Ports{
		Underlay:      c.spineUplinks,
		Unprovisioned: []string{},
		Vrfs:          map[string]*types.Vrf{},
		Firewalls:     map[string]*types.Firewall{},
		DownPorts:     map[string]bool{},
	}
	p.BladePorts = c.additionalBridgePorts
	for _, nic := range s.Nics {
		port := *nic.Name

		if isPortStatusEqual(models.V1SwitchNicActualDOWN, nic.Actual) {
			if has := p.DownPorts[port]; !has {
				p.DownPorts[port] = true
			}
		}

		if slices.Contains(p.Underlay, port) {
			continue
		}
		if slices.Contains(c.additionalBridgePorts, port) {
			continue
		}
		if nic.Vrf == "" {
			if !slices.Contains(p.Unprovisioned, port) {
				p.Unprovisioned = append(p.Unprovisioned, port)
			}
			continue
		}

		// Firewall-Port
		if nic.Vrf == "default" {
			fw := &types.Firewall{
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
		vrf := &types.Vrf{}
		if v, has := p.Vrfs[nic.Vrf]; has {
			vrf = v
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(nic.Vrf, "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		vrf.VNI = uint32(vni64) // nolint:gosec
		vrf.Neighbors = append(vrf.Neighbors, port)
		if nic.Filter != nil {
			vrf.Cidrs = nic.Filter.Cidrs
		}
		p.Vrfs[nic.Vrf] = vrf
	}
	switcherConfig.Ports = p

	c.nos.SanitizeConfig(switcherConfig)
	err = switcherConfig.FillRouteMapsAndIPPrefixLists()
	if err != nil {
		return nil, err
	}
	m, err := vlan.ReadMapping()
	if err != nil {
		return nil, err
	}
	err = switcherConfig.FillVLANIDs(m)
	if err != nil {
		return nil, err
	}

	return switcherConfig, nil
}

// mapLogLevel maps the metal-core log level to an appropriate FRR log level
// http://docs.frrouting.org/en/latest/basic.html#clicmd-[no]logsyslog[LEVEL]
func mapLogLevel(level string) string {
	switch strings.ToLower(level) {
	case "debug":
		return "debugging"
	case "info":
		return "informational"
	case "warn":
		return "warnings"
	case "error":
		return "errors"
	default:
		return "warnings"
	}
}

func fillEth0Info(c *types.Conf, gw string) error {
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
	s, _ := addrs[0].IPNet.Mask.Size()
	c.Ports.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Ports.Eth0.Gateway = gw
	return nil
}

// isLinkUp checks if the interface with the given name is up.
// It returns a boolean indicating if the interface is up, and an error if there was a problem checking the interface.
func isLinkUp(nicname string) (bool, error) {
	nic, err := net.InterfaceByName(nicname)
	if err != nil {
		return false, fmt.Errorf("cannot query interface %q : %w", nicname, err)
	}
	return nic.Flags&net.FlagUp != 0, nil
}

func isPortStatusEqual(stat string, other *string) bool {
	if other == nil {
		return false
	}
	return strings.EqualFold(stat, *other)
}
