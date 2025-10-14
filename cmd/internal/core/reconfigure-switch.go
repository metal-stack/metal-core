package core

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vishvananda/netlink"
	"google.golang.org/protobuf/types/known/durationpb"

	adminv2 "github.com/metal-stack/api/go/metalstack/admin/v2"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-core/cmd/internal/frr"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

// ConstantlyReconfigureSwitch reconfigures the switch.
func (c *Core) ConstantlyReconfigureSwitch(ctx context.Context, interval time.Duration) {
	host, _ := os.Hostname()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.log.Info("trigger reconfiguration")
			start := time.Now()
			s, err := c.reconfigureSwitch(ctx, host)
			elapsed := time.Since(start)
			c.log.Info("reconfiguration took", "elapsed", elapsed)

			req := (&infrav2.SwitchServiceHeartbeatRequest{
				Id:            host,
				Duration:      durationpb.New(time.Duration(elapsed.Nanoseconds())),
				PortStates:    map[string]apiv2.SwitchPortStatus{},
				BgpPortStates: map[string]*apiv2.SwitchBGPPortState{},
			})

			if err != nil {
				req.Error = pointer.Pointer(err.Error())
				c.log.Error("reconfiguration failed", "error", err)
				c.metrics.CountError("switch-reconfiguration")
			} else {
				c.log.Info("reconfiguration succeeded")
			}

			var nics []*apiv2.SwitchNic
			if s != nil {
				nics = s.Nics
			}
			for _, nic := range nics {
				if nic == nil || nic.Name == "" {
					c.log.Error("could not check if link is up", "nic", nic)
					c.metrics.CountError("switch-reconfiguration")
					continue
				}
				isup, err := isLinkUp(nic.Name)
				if err != nil {
					c.log.Error("could not check if link is up", "error", err, "nicname", nic.Name)
					req.PortStates[nic.Name] = apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UNKNOWN
					c.metrics.CountError("switch-reconfiguration")
					continue
				}
				if isup {
					req.PortStates[nic.Name] = apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP
				} else {
					req.PortStates[nic.Name] = apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN
				}
			}

			if c.bgpNeighborStateFile != "" {
				bgpportstates, err := frr.GetBGPStates(c.bgpNeighborStateFile)
				if err != nil {
					c.log.Error("could not get BGP states", "error", err)
					c.metrics.CountError("switch-reconfiguration")
				}
				req.BgpPortStates = bgpportstates
			}

			_, err = c.client.Infrav2().Switch().Heartbeat(ctx, connect.NewRequest(req))
			if err != nil {
				c.log.Error("notification about switch reconfiguration failed", "error", err)
				c.metrics.CountError("reconfiguration-notification")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Core) reconfigureSwitch(ctx context.Context, hostname string) (*apiv2.Switch, error) {
	req := &adminv2.SwitchServiceGetRequest{
		Id: hostname,
	}

	res, err := c.client.Adminv2().Switch().Get(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch switch: %w", err)
	}

	s := res.Msg.Switch
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

func (c *Core) buildSwitcherConfig(s *apiv2.Switch) (*types.Conf, error) {
	asn64, err := strconv.ParseUint(c.asn, 10, 32)
	if err != nil {
		return nil, err
	}
	if c.pxeVlanID >= vlan.VlanIDMin && c.pxeVlanID <= vlan.VlanIDMax {
		return nil, fmt.Errorf("configured PXE VLAN ID is in the reserved area of %d, %d", vlan.VlanIDMin, vlan.VlanIDMax)
	}

	switcherConfig := &types.Conf{
		Name:                 s.Id,
		LogLevel:             mapLogLevel(c.logLevel),
		ASN:                  uint32(asn64), // nolint:gosec
		Loopback:             c.loopbackIP,
		MetalCoreCIDR:        c.cidr,
		AdditionalBridgeVIDs: c.additionalBridgeVIDs,
		PXEVlanID:            c.pxeVlanID,
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
		port := nic.Name

		if nic.State != nil && nic.State.Actual == apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN {
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
		if pointer.SafeDeref(nic.Vrf) == "" {
			if !slices.Contains(p.Unprovisioned, port) {
				p.Unprovisioned = append(p.Unprovisioned, port)
			}
			continue
		}

		// Firewall-Port
		if pointer.SafeDeref(nic.Vrf) == "default" {
			fw := &types.Firewall{
				Port: port,
			}
			if nic.BgpFilter != nil {
				fw.Vnis = nic.BgpFilter.Vnis
				fw.Cidrs = nic.BgpFilter.Cidrs
			}
			p.Firewalls[port] = fw
			continue
		}

		// Machine-Port
		vrf := &types.Vrf{}
		if v, has := p.Vrfs[pointer.SafeDeref(nic.Vrf)]; has {
			vrf = v
		}
		vni64, err := strconv.ParseUint(strings.TrimPrefix(pointer.SafeDeref(nic.Vrf), "vrf"), 10, 32)
		if err != nil {
			return nil, err
		}
		vrf.VNI = uint32(vni64) // nolint:gosec
		vrf.Neighbors = append(vrf.Neighbors, port)
		if nic.BgpFilter != nil {
			vrf.Cidrs = nic.BgpFilter.Cidrs
		}
		p.Vrfs[pointer.SafeDeref(nic.Vrf)] = vrf
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
	s, _ := addrs[0].Mask.Size()
	c.Ports.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Ports.Eth0.Gateway = gw
	return nil
}

func isLinkUp(nicname string) (bool, error) {
	nic, err := net.InterfaceByName(nicname)
	if err != nil {
		return false, fmt.Errorf("cannot query interface %q : %w", nicname, err)
	}
	return nic.Flags&net.FlagUp != 0, nil
}
