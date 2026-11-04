package core

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/vishvananda/netlink"
	"google.golang.org/protobuf/types/known/durationpb"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-core/cmd/internal/frr"
	corenet "github.com/metal-stack/metal-core/cmd/internal/net"
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
				status, err := corenet.GetLinkStatus(nic.Name)
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
	const (
		defaultVRF        = "default"
		peerGroupFabric   = "FABRIC"
		peerGroupFirewall = "FIREWALL"
		peerGroupMachine  = "MACHINE"
	)

	var (
		err                  error
		l3vnis               []uint32
		unnumberedInterfaces []*types.Interface
		bgps                 []*types.BGP
		routeMaps            []*types.RouteMap
		prefixLists          []*types.IPPrefixList
	)

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
	}

	for _, nic := range s.Nics {
		if nic == nil {
			continue
		}

		if slices.Contains(c.ignorePorts, nic.Name) {
			continue
		}

		l3vnis, err = addL3VNI(nic.Vrf, l3vnis)
		if err != nil {
			return nil, fmt.Errorf("failed to add l3vni of port %s: %w", nic.Name, err)
		}

		unnumberedInterface := getUnnumberedInterface(nic.Name, c.nos.GetVRF(nic), c.spineUplinks)
		if unnumberedInterface == nil {
			// FIXME: this is an unprovisioned port then; treat accordingly
			continue
		}
		unnumberedInterfaces = append(unnumberedInterfaces, unnumberedInterface)

		bgp, found := lo.Find(bgps, func(b *types.BGP) bool {
			return pointer.SafeDeref(b.Vrf) == pointer.SafeDeref(unnumberedInterface.Vrf)
		})

		if !found {
			bgp = &types.BGP{
				Vrf: unnumberedInterface.Vrf,
			}
		}

		if bgp.Vrf == nil {
			addUnderlayBGPPort(unnumberedInterface, bgp, c.spineUplinks)
		} else {
			err := addMachineBGPPort(*unnumberedInterface, nic.BgpFilter, bgp)
			if err != nil {
				return nil, fmt.Errorf("failed to add bgp session for machine port %s: %w", nic.Name, err)
			}
		}

		if !found {
			bgps = append(bgps, bgp)
		}

		if nic.BgpFilter == nil {
			continue
		}

		if len(nic.BgpFilter.Cidrs) > 0 {
			var rmName string
			if unnumberedInterface.Vrf == nil {
				rmName = fmt.Sprintf("fw-%s-in", unnumberedInterface.Name)
			} else {
				rmName = fmt.Sprintf("%s-in", pointer.SafeDeref(unnumberedInterface.Vrf))
			}

			rms, pfs, err := getCIDRRouteMapsAndPrefixLists(rmName, nic.BgpFilter.Cidrs)
			if err != nil {
				return nil, fmt.Errorf("failed to make route-maps and prefix-lists for port %s: %w", nic.Name, err)
			}

			routeMaps = append(routeMaps, rms...)
			prefixLists = append(prefixLists, pfs...)
		}

		if len(nic.BgpFilter.Vnis) > 0 {
			rms := getVNIRouteMaps(fmt.Sprintf("fw-%s-vni", unnumberedInterface.Name), nic.BgpFilter.Vnis)
			routeMaps = append(routeMaps, rms...)
		}

		// port := &types.Port{
		// 	Vrf: nic.Vrf,
		// 	MTU: 9000,
		// }

		// // TODO: discuss if it should be allowed to set `admin_status: down` for spine uplinks
		// adminStatus := pointer.SafeDeref(pointer.SafeDeref(nic.State).Desired)
		// switch adminStatus {
		// case apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN:
		// 	port.AdminStatus = types.PortStatusDown
		// case apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP:
		// 	port.AdminStatus = types.PortStatusUp
		// default:
		// 	// noop
		// }

		// switch {
		// case pointer.SafeDeref(port.Vrf) == defaultVRF:
		// 	port.MTU = 9216
		// 	port.PeerGroup = new(peerGroupFirewall)
		// 	port.Routed = true
		// case slices.Contains(c.spineUplinks, nic.Name):
		// 	port.MTU = 9216
		// 	port.PeerGroup = new(peerGroupFabric)
		// 	port.Routed = true
		// 	port.Vrf = new(defaultVRF)
		// case port.Vrf != nil:
		// 	port.PeerGroup = new(peerGroupMachine)
		// 	port.Routed = true
		// default:
		// 	port.VLAN = &c.pxeVlanID
		// 	switcherConfig.Ports[nic.Name] = port
		// 	continue
		// }

		// if nic.BgpFilter != nil {
		// 	port.Cidrs = nic.BgpFilter.Cidrs
		// 	port.Vnis = nic.BgpFilter.Vnis
		// }

		// vrf := pointer.SafeDeref(port.Vrf)
		// session, ok := switcherConfig.BGPSessions[vrf]
		// if !ok {
		// 	has4, has6, err := port.HasAddressFamilies()
		// 	if err != nil {
		// 		return nil, fmt.Errorf("failed to check address families for port %s: %w", nic.Name, err)
		// 	}

		// 	session = &types.BGPSession{
		// 		Vrf: &types.Vrf{
		// 			Has4: has4,
		// 			Has6: has6,
		// 		},
		// 		PeerGroups: types.PeerGroups{},
		// 	}
		// }

		// pgName := pointer.SafeDeref(port.PeerGroup)
		// peerGroup, ok := session.PeerGroups[pgName]
		// if !ok {
		// 	peerGroup = &types.PeerGroup{}
		// }

		// peerGroup.Neighbors = append(peerGroup.Neighbors, fmt.Sprintf("%s interface", nic.Name))
		// if vrf == defaultVRF && peerGroup.Commands == nil {
		// 	commands := &types.Commands{
		// 		Global: []string{
		// 			"remote-as external",
		// 			"timers 2 8",
		// 		},
		// 		EVPN: []string{
		// 			"activate",
		// 			"allowas-in 2",
		// 		},
		// 	}

		// 	if pgName == "FIREWALL" {
		// 		if session.Vrf != nil && session.Vrf.Has4 {
		// 			commands.IPv4 = []string{
		// 				"allowas-in 2",
		// 			}
		// 		}
		// 		if session.Vrf != nil && session.Vrf.Has6 {
		// 			commands.IPv6 = []string{
		// 				"activate",
		// 				"allowas-in 2",
		// 			}
		// 		}
		// 	}

		// 	peerGroup.Commands = commands
		// }

		// session.PeerGroups[pgName] = peerGroup
		// switcherConfig.BGPSessions[vrf] = session
		// switcherConfig.Ports[nic.Name] = port
	}

	slices.SortFunc(bgps, func(a, b *types.BGP) int {
		return strings.Compare(pointer.SafeDeref(a.Vrf), pointer.SafeDeref(b.Vrf))
	})

	switcherConfig.FRRConfig = &types.FRRConfig{
		L3VNIs:               l3vnis,
		UnnumberedInterfaces: unnumberedInterfaces,
		BGP:                  bgps,
		RouteMaps:            routeMaps,
		PrefixLists:          prefixLists,
	}

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

func getUnnumberedInterface(nicName string, vrf *string, underlayPorts []string) *types.Interface {
	if slices.Contains(underlayPorts, nicName) {
		return &types.Interface{
			Name: nicName,
		}
	}

	if vrf == nil {
		return nil
	}

	if pointer.SafeDeref(vrf) == "default" {
		return &types.Interface{
			Name: nicName,
		}
	}

	return &types.Interface{
		Name: nicName,
		Vrf:  vrf,
	}
}

func addL3VNI(vrf *string, vnis []uint32) ([]uint32, error) {
	if vrf == nil {
		return vnis, nil
	}

	if pointer.SafeDeref(vrf) == "default" {
		return vnis, nil
	}

	vniString, _ := strings.CutPrefix(pointer.SafeDeref(vrf), "vrf")
	vni, err := strconv.ParseUint(vniString, 10, 32)
	if err != nil {
		return vnis, fmt.Errorf("failed to parse vni of vrf %s: %w", pointer.SafeDeref(vrf), err)
	}

	if !slices.Contains(vnis, uint32(vni)) {
		vnis = append(vnis, uint32(vni))
	}

	return vnis, nil
}

func getPeerGroups(groups []string) []types.PeerGroup {
	var peerGroups []types.PeerGroup

	for _, group := range groups {
		peerGroups = append(peerGroups, types.PeerGroup{
			Name: group,
			Commands: []string{
				"remote-as external",
				"timers 2 8",
			},
		})
	}

	return peerGroups
}

func addUnderlayBGPPort(intf *types.Interface, bgp *types.BGP, underlayPorts []string) {
	if bgp.PeerGroups == nil {
		bgp.PeerGroups = getPeerGroups([]string{"FABRIC", "FIREWALL"})
	}

	neigh := types.BGPNeighbor{
		Name: fmt.Sprintf("%s interface", intf.Name),
	}

	if slices.Contains(underlayPorts, intf.Name) {
		neigh.PeerGroup = new("FABRIC")
	} else {
		neigh.PeerGroup = new("FIREWALL")
	}

	bgp.Neighbors = append(bgp.Neighbors, neigh)

	if bgp.AddressFamilies.IPv4Unicast == nil {
		bgp.AddressFamilies.IPv4Unicast = &types.BGPAddressFamily{
			AddressFamily: types.BGPAddressFamilyTypeIPv4,
			GlobalStatements: []string{
				"redistribute connected route-map DENY_MGMT",
				"redistribute kernel route-map DENY_MGMT",
			},
			Neighbors: []types.AdressFamilyNeighbor{
				{
					Neighbor: "FIREWALL",
					Commands: []string{"allowas-in 2"},
				},
			},
		}
	}

	if bgp.AddressFamilies.IPv6Unicast == nil {
		bgp.AddressFamilies.IPv6Unicast = &types.BGPAddressFamily{
			AddressFamily: types.BGPAddressFamilyTypeIPv6,
			GlobalStatements: []string{
				"redistribute connected route-map DENY_MGMT",
				`redistribute kernel route-map DENY_MGMT`,
			},
			Neighbors: []types.AdressFamilyNeighbor{
				{
					Neighbor: "FIREWALL",
					Commands: []string{"activate", "allowas-in 2"},
				},
			},
		}
	}

	if bgp.AddressFamilies.L2VPNEVPN == nil {
		bgp.AddressFamilies.L2VPNEVPN = &types.BGPAddressFamily{
			AddressFamily:    types.BGPAddressFamilyTypeEVPN,
			GlobalStatements: []string{"advertise-all-vni"},
			Neighbors: []types.AdressFamilyNeighbor{
				{
					Neighbor: "FABRIC",
					Commands: []string{"activate", "allowas-in 2"},
				},
				{
					Neighbor: "FIREWALL",
					Commands: []string{"activate", "allowas-in 2"},
				},
			},
		}
	}

	if !slices.Contains(underlayPorts, intf.Name) {
		neighs := bgp.AddressFamilies.IPv4Unicast.Neighbors
		neighs = append(neighs, types.AdressFamilyNeighbor{
			Neighbor:   intf.Name,
			RouteMapIn: fmt.Sprintf("fw-%s-in", intf.Name),
		})
		bgp.AddressFamilies.IPv4Unicast.Neighbors = neighs

		neighs = bgp.AddressFamilies.IPv6Unicast.Neighbors
		neighs = append(neighs, types.AdressFamilyNeighbor{
			Neighbor:   intf.Name,
			RouteMapIn: fmt.Sprintf("fw-%s-in", intf.Name),
		})
		bgp.AddressFamilies.IPv6Unicast.Neighbors = neighs

		neighs = bgp.AddressFamilies.L2VPNEVPN.Neighbors
		neighs = append(neighs, types.AdressFamilyNeighbor{
			Neighbor:    intf.Name,
			RouteMapOut: fmt.Sprintf("fw-%s-vni", intf.Name),
		})
		bgp.AddressFamilies.L2VPNEVPN.Neighbors = neighs
	}
}

func addMachineBGPPort(intf types.Interface, bgpFilter *apiv2.BGPFilter, bgp *types.BGP) error {
	if bgp.PeerGroups == nil {
		bgp.PeerGroups = getPeerGroups([]string{"MACHINE"})
	}

	neigh := types.BGPNeighbor{
		Name:      fmt.Sprintf("%s interface", intf.Name),
		PeerGroup: new("MACHINE"),
	}

	bgp.Neighbors = append(bgp.Neighbors, neigh)

	var (
		cidrs []string
	)

	if bgpFilter != nil {
		cidrs = bgpFilter.Cidrs
	}

	has4, err := hasAddressFamily(cidrs, types.AddressFamilyIPv4)
	if err != nil {
		return fmt.Errorf("failed to check if port %s has ipv4 addresses: %w", intf.Name, err)
	}

	if has4 && bgp.AddressFamilies.IPv4Unicast == nil {
		bgp.AddressFamilies.IPv4Unicast = &types.BGPAddressFamily{
			AddressFamily: types.BGPAddressFamilyTypeIPv4,
			GlobalStatements: []string{
				"redistribute connected",
			},
			Neighbors: []types.AdressFamilyNeighbor{
				{
					Neighbor:   "MACHINE",
					Commands:   []string{"maximum-prefix 24000"},
					RouteMapIn: fmt.Sprintf("%s-in", pointer.SafeDeref(intf.Vrf)),
				},
			},
		}
	}

	has6, err := hasAddressFamily(cidrs, types.AddressFamilyIPv6)
	if err != nil {
		return fmt.Errorf("failed to check if port %s has ipv6 addresses: %w", intf.Name, err)
	}

	if has6 && bgp.AddressFamilies.IPv6Unicast == nil {
		bgp.AddressFamilies.IPv6Unicast = &types.BGPAddressFamily{
			AddressFamily: types.BGPAddressFamilyTypeIPv6,
			GlobalStatements: []string{
				"redistribute connected",
			},
			Neighbors: []types.AdressFamilyNeighbor{
				{
					Neighbor:   "MACHINE",
					Commands:   []string{"activate", "maximum-prefix 24000"},
					RouteMapIn: fmt.Sprintf("%s-in6", pointer.SafeDeref(intf.Vrf)),
				},
			},
		}
	}

	if bgp.AddressFamilies.L2VPNEVPN == nil {
		var globalStatements []string
		if has4 {
			globalStatements = append(globalStatements, "advertise ipv4 unicast")
		}
		if has6 {
			globalStatements = append(globalStatements, "advertise ipv6 unicast")
		}

		bgp.AddressFamilies.L2VPNEVPN = &types.BGPAddressFamily{
			AddressFamily:    types.BGPAddressFamilyTypeEVPN,
			GlobalStatements: globalStatements,
		}
	}

	return nil
}

func getCIDRRouteMapsAndPrefixLists(rmName string, cidrs []string) ([]*types.RouteMap, []*types.IPPrefixList, error) {
	var (
		prefixLists []*types.IPPrefixList
		routeMaps   []*types.RouteMap
		has4        bool
		has6        bool
	)

	for _, cidr := range cidrs {
		addr, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse cidr %s: %w", cidr, err)
		}

		var (
			af     = types.AddressFamilyIPv4
			pfName string
		)
		if addr.Addr().Is4() {
			has4 = true
			pfName = fmt.Sprintf("%s-prefixes", rmName)
		}
		if addr.Addr().Is6() {
			af = types.AddressFamilyIPv6
			has6 = true
			pfName = fmt.Sprintf("%s6-prefixes", rmName)
		}

		prefixLists = append(prefixLists, &types.IPPrefixList{
			AddressFamily: af,
			Name:          pfName,
			Spec:          fmt.Sprintf("permit %s le %d", cidr, addr.Addr().BitLen()),
		})
	}

	if has4 {
		routeMaps = append(routeMaps, &types.RouteMap{
			Name:    rmName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s-prefixes", rmName)},
			Policy:  types.PolicyPermit,
			Order:   10,
		})
	}

	if has6 {
		routeMaps = append(routeMaps, &types.RouteMap{
			Name:    fmt.Sprintf("%s6", rmName),
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s6-prefixes", rmName)},
			Policy:  types.PolicyPermit,
			Order:   10,
		})
	}

	return routeMaps, prefixLists, nil
}

func getVNIRouteMaps(rmName string, vnis []string) []*types.RouteMap {
	var routeMaps []*types.RouteMap

	for i, vni := range vnis {
		routeMaps = append(routeMaps, &types.RouteMap{
			Name:    rmName,
			Entries: []string{fmt.Sprintf("match evpn vni %s", vni)},
			Policy:  types.PolicyPermit,
			Order:   10 + i,
		})
	}

	return routeMaps
}

func hasAddressFamily(cidrs []string, af types.AddressFamily) (bool, error) {
	var (
		has4 bool
		has6 bool
	)

	for _, cidr := range cidrs {
		addr, err := netip.ParsePrefix(cidr)
		if err != nil {
			return false, fmt.Errorf("failed to parse cidr %s: %w", cidr, err)
		}

		if addr.Addr().Is4() {
			has4 = true
		}
		if addr.Addr().Is6() {
			has6 = true
		}
	}

	if af == types.AddressFamilyIPv4 {
		return has4, nil
	} else {
		return has6, nil
	}
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
