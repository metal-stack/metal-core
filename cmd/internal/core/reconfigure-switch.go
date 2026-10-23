package core

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"google.golang.org/protobuf/types/known/durationpb"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-core/cmd/internal/frr"
	"github.com/metal-stack/metal-core/cmd/internal/net"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

func (c *Core) ConstantlyReconfigureSwitch(ctx context.Context, interval, timeout time.Duration) {
	host, _ := os.Hostname()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			withTimeout, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			c.log.Info("trigger reconfiguration")
			start := time.Now()
			s, err := c.reconfigureSwitch(withTimeout, host)
			elapsed := time.Since(start)
			c.log.Info("reconfiguration took", "elapsed", elapsed)

			req := &infrav2.SwitchServiceHeartbeatRequest{
				Id:            host,
				Duration:      durationpb.New(time.Duration(elapsed.Nanoseconds())),
				PortStates:    map[string]apiv2.SwitchPortStatus{},
				BgpPortStates: map[string]*apiv2.SwitchBGPPortState{},
			}

			if err != nil {
				req.Error = new(err.Error())
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
				if nic == nil {
					continue
				}

				if nic.Name == "" {
					c.log.Error("skipping nic because name is empty", "nic", nic)
					c.metrics.CountError("switch-reconfiguration")
					continue
				}
				status, err := net.GetLinkStatus(nic.Name)
				req.PortStates[nic.Name] = status
				if err != nil {
					c.log.Error("could not check if link is up", "error", err, "nicname", nic.Name)
					c.metrics.CountError("switch-reconfiguration")
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

			_, err = c.client.Infrav2().Switch().Heartbeat(ctx, req)
			if err != nil {
				c.log.Error("switch heartbeat failed", "error", err)
				c.metrics.CountError("reconfiguration-notification")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Core) reconfigureSwitch(ctx context.Context, hostname string) (*apiv2.Switch, error) {
	req := &infrav2.SwitchServiceGetRequest{
		Id: hostname,
	}

	res, err := c.client.Infrav2().Switch().Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch switch: %w", err)
	}

	s := res.Switch
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

	err = c.nos.Apply(ctx, switchConfig)
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
		AdditionalMgmtRoutes: c.additionalMgmtRoutes,
		PXEVlanID:            c.pxeVlanID,
		SetSrcLoopback:       c.setSrcLoopback,
		Ports:                types.Ports{},
		Vrfs:                 types.Vrfs{},
		BGPSessions:          types.BGPSessions{},
	}

	for _, nic := range s.Nics {
		if nic == nil {
			continue
		}

		if slices.Contains(c.ignorePorts, nic.Name) {
			continue
		}

		port := &types.Port{
			Vrf: nic.Vrf,
		}

		// TODO: discuss if it should be allowed to set `admin_status: down` for spine uplinks
		adminStatus := pointer.SafeDeref(pointer.SafeDeref(nic.State).Desired)
		switch adminStatus {
		case apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN:
			port.AdminStatus = types.PortStatusDown
		case apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP:
			port.AdminStatus = types.PortStatusUp
		default:
			// noop
		}

		// FIXME: too many implicit assumptions; try to make everything more explicit
		switch pointer.SafeDeref(port.Vrf) {
		case "default":
			port.MTU = 9216
			port.Routed = true
			port.PeerGroup = new("FIREWALL")
		case "":
			port.MTU = 900
			port.VLAN = &c.pxeVlanID
			switcherConfig.Ports[nic.Name] = port
			continue
		default:
			port.MTU = 9000
			port.Routed = true
			port.PeerGroup = new("MACHINE")
		}

		if slices.Contains(c.spineUplinks, nic.Name) {
			port.MTU = 9216
			port.Routed = true
			port.PeerGroup = new("FABRIC")
		}

		// CONTINUE:
		// - populate bgp sessions for frr config

		vrf := pointer.SafeDeref(port.Vrf)
		// peerGroup := "MACHINE"
		// if vrf == "default" {
		// 	peerGroup = "FIREWALL"
		// }

		session, ok := switcherConfig.BGPSessions[vrf]
		if !ok {
			session = &types.BGPSession{}
		}

		switcherConfig.BGPSessions[vrf] = session
		switcherConfig.Ports[nic.Name] = port
	}

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
	c.Eth0 = types.MgmtInterface{}
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
	c.Eth0.AddressCIDR = fmt.Sprintf("%s/%d", ip.String(), s)
	c.Eth0.Gateway = gw
	return nil
}
