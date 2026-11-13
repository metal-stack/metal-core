package core

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
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

var interfaceNumbers = regexp.MustCompile("[0-9]+")

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
	var (
		err                  error
		vrfs                 []string
		unnumberedInterfaces []*types.Interface
		bgps                 []*types.BGP
		routeMaps            = c.nos.DefaultRouteMaps()
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

		if c.staticFRRConfig != nil && isPortStaticNeighbor(nic.Name, c.staticFRRConfig.BGPs) {
			continue
		}

		port := types.Port{
			Name: nic.Name,
			MTU:  9000,
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

		unnumberedInterface := getUnnumberedInterface(nic.Name, c.nos.GetVRF(nic), c.spineUplinks)
		if unnumberedInterface == nil {
			port.VLAN = new(fmt.Sprintf("Vlan%d", c.pxeVlanID))
			switcherConfig.Ports = append(switcherConfig.Ports, port)
			continue
		}
		unnumberedInterfaces = append(unnumberedInterfaces, unnumberedInterface)

		switch {
		case unnumberedInterface.Vrf == nil, pointer.SafeDeref(unnumberedInterface.Vrf) == "default":
			port.MTU = 9216
		default:
			port.VRF = unnumberedInterface.Vrf
		}
		switcherConfig.Ports = append(switcherConfig.Ports, port)

		vrfs = addVRFtoFRRConfig(unnumberedInterface.Vrf, vrfs)
		switcherConfig.Vrfs, err = addVRF(unnumberedInterface.Vrf, switcherConfig.Vrfs)
		if err != nil {
			return nil, fmt.Errorf("failed to add vrf of port %s: %w", nic.Name, err)
		}

		bgp, found := lo.Find(bgps, func(b *types.BGP) bool {
			return pointer.SafeDeref(b.Vrf) == pointer.SafeDeref(unnumberedInterface.Vrf)
		})

		if !found {
			bgp = &types.BGP{
				Vrf: unnumberedInterface.Vrf,
			}
		}

		if bgp.Vrf == nil {
			addUnderlayBGPPort(unnumberedInterface, bgp, c.spineUplinks, c.nos.DefaultRouteMaps()[0].Name)
		} else {
			err := addMachineBGPPort(*unnumberedInterface, nic.BgpFilter, bgp)
			if err != nil {
				return nil, fmt.Errorf("failed to add bgp session for machine port %s: %w", nic.Name, err)
			}
		}

		slices.SortFunc(bgp.Neighbors, compareBGPNeighbors)

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
	}

	if c.staticFRRConfig != nil {
		for _, vrf := range c.staticFRRConfig.VRFs {
			switcherConfig.Vrfs, err = addVRF(&vrf, switcherConfig.Vrfs)
			if err != nil {
				return nil, fmt.Errorf("failed to add vrf %s to config: %w", vrf, err)
			}
		}
	}

	slices.SortFunc(switcherConfig.Vrfs, func(a, b *types.Vrf) int {
		return strings.Compare(pointer.SafeDeref(a).Name, pointer.SafeDeref(b).Name)
	})

	frrConfig := &types.FRRConfig{
		VRFs:                 vrfs,
		UnnumberedInterfaces: unnumberedInterfaces,
		BGPs:                 bgps,
		RouteMaps:            routeMaps,
		PrefixLists:          prefixLists,
	}

	mergedConfig, err := mergeFRRConfigs(c.staticFRRConfig, frrConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to merge static and dynamic frr configs: %w", err)
	}

	switcherConfig.FRRConfig = mergedConfig

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

func mergeFRRConfigs(conf1, conf2 *types.FRRConfig) (*types.FRRConfig, error) {
	if conf1 == nil {
		return conf2, nil
	}
	if conf2 == nil {
		return conf1, nil
	}

	conf := &types.FRRConfig{
		VRFs:        mergeVRFs(conf1.VRFs, conf2.VRFs),
		PrefixLists: mergePrefixLists(conf1.PrefixLists, conf2.PrefixLists),
	}

	unnumberedInterfaces, err := mergeInterfaces(conf1.UnnumberedInterfaces, conf2.UnnumberedInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to merge unnumbered interfaces: %w", err)
	}
	conf.UnnumberedInterfaces = unnumberedInterfaces

	bgps, err := mergeBGPs(conf1.BGPs, conf2.BGPs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge bgp configs: %w", err)
	}
	conf.BGPs = bgps

	routeMaps, err := mergeRouteMaps(conf1.RouteMaps, conf2.RouteMaps)
	if err != nil {
		return nil, fmt.Errorf("failed to merge route-maps: %w", err)
	}
	conf.RouteMaps = routeMaps

	return conf, nil
}

func mergeVRFs(vrfs1, vrfs2 []string) []string {
	var vrfs []string

	for _, vrf := range append(vrfs1, vrfs2...) {
		if slices.Contains(vrfs, vrf) {
			continue
		}
		vrfs = append(vrfs, vrf)
	}
	slices.Sort(vrfs)

	return vrfs
}

func mergeInterfaces(ifs1, ifs2 []*types.Interface) ([]*types.Interface, error) {
	var ifs []*types.Interface

	for _, intf := range append(ifs1, ifs2...) {
		if slices.ContainsFunc(ifs, func(i *types.Interface) bool {
			return i.Name == intf.Name
		}) {
			return nil, fmt.Errorf("found duplicate interface %s", intf.Name)
		}
		ifs = append(ifs, intf)
	}
	slices.SortFunc(ifs, func(a, b *types.Interface) int {
		return interfaceSmallerFunc(a.Name, b.Name)
	})

	return ifs, nil
}

func mergeBGPs(bgps1, bgps2 []*types.BGP) ([]*types.BGP, error) {
	var bgps []*types.BGP

	for _, bgp := range append(bgps1, bgps2...) {
		existingBGP, found := lo.Find(bgps, func(b *types.BGP) bool {
			return pointer.SafeDeref(b.Vrf) == pointer.SafeDeref(bgp.Vrf)
		})

		if !found {
			bgps = append(bgps, bgp)
			continue
		}

		pgs, err := mergePeerGroups(existingBGP.PeerGroups, bgp.PeerGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to merge bgp configs for vrf %s: %w", pointer.SafeDeref(bgp.Vrf), err)
		}
		existingBGP.PeerGroups = pgs

		neighs, err := mergeBGPNeighbors(existingBGP.Neighbors, bgp.Neighbors)
		if err != nil {
			return nil, fmt.Errorf("failed to merge bgp neighbors for vrf %s: %w", pointer.SafeDeref(bgp.Vrf), err)
		}
		existingBGP.Neighbors = neighs

		existingBGP.AddressFamilies = types.BGPAddressFamilies{
			IPv4Unicast: mergeBGPAddressFamily(existingBGP.AddressFamilies.IPv4Unicast, bgp.AddressFamilies.IPv4Unicast),
			IPv6Unicast: mergeBGPAddressFamily(existingBGP.AddressFamilies.IPv6Unicast, bgp.AddressFamilies.IPv6Unicast),
			L2VPNEVPN:   mergeBGPAddressFamily(existingBGP.AddressFamilies.L2VPNEVPN, bgp.AddressFamilies.L2VPNEVPN),
		}
	}

	slices.SortFunc(bgps, func(a, b *types.BGP) int {
		return strings.Compare(pointer.SafeDeref(a.Vrf), pointer.SafeDeref(b.Vrf))
	})

	return bgps, nil
}

func mergeBGPNeighbors(neighs1, neighs2 []types.BGPNeighbor) ([]types.BGPNeighbor, error) {
	var neighs []types.BGPNeighbor

	for _, neigh := range append(neighs1, neighs2...) {
		if slices.ContainsFunc(neighs, func(n types.BGPNeighbor) bool {
			return n.Name == neigh.Name
		}) {
			return nil, fmt.Errorf("cannot add neighbor %q twice", neigh.Name)
		}
		neighs = append(neighs, neigh)
	}

	slices.SortFunc(neighs, compareBGPNeighbors)

	return neighs, nil
}

func compareBGPNeighbors(a, b types.BGPNeighbor) int {
	pgDiff := strings.Compare(pointer.SafeDeref(a.PeerGroup), pointer.SafeDeref(b.PeerGroup))
	if pgDiff != 0 {
		return pgDiff
	}
	aIfname, aFound := strings.CutSuffix(a.Name, " interface")
	bIfname, bFound := strings.CutSuffix(b.Name, " interface")
	if aFound && bFound {
		return interfaceSmallerFunc(aIfname, bIfname)
	}
	return strings.Compare(a.Name, b.Name)
}

func mergeBGPAddressFamily(af1, af2 *types.BGPAddressFamily) *types.BGPAddressFamily {
	af := &types.BGPAddressFamily{}

	if af1 == nil {
		return af2
	}
	if af2 == nil {
		return af1
	}

	for _, cmd := range append(af1.GlobalCommands, af2.GlobalCommands...) {
		if !slices.Contains(af.GlobalCommands, cmd) {
			af.GlobalCommands = append(af.GlobalCommands, cmd)
		}
	}

	for _, neigh := range append(af1.Neighbors, af2.Neighbors...) {
		existing, found := lo.Find(af.Neighbors, func(n *types.AddressFamilyNeighbor) bool {
			return n.Name == neigh.Name
		})
		if !found {
			af.Neighbors = append(af.Neighbors, neigh)
			continue
		}

		for _, cmd := range append(neigh.Commands, existing.Commands...) {
			if !slices.Contains(existing.Commands, cmd) {
				existing.Commands = append(existing.Commands, cmd)
			}
		}
	}

	slices.SortFunc(af.Neighbors, func(a, b *types.AddressFamilyNeighbor) int {
		return strings.Compare(a.Name, b.Name)
	})

	return af
}

func mergePeerGroups(pgs1, pgs2 []types.PeerGroup) ([]types.PeerGroup, error) {
	var pgs []types.PeerGroup

	for _, pg := range append(pgs1, pgs2...) {
		if slices.ContainsFunc(pgs, func(p types.PeerGroup) bool {
			return p.Name == pg.Name
		}) {
			return nil, fmt.Errorf("cannot add peer-group %q twice", pg.Name)
		}

		pgs = append(pgs, pg)
	}

	slices.SortFunc(pgs, func(a, b types.PeerGroup) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pgs, nil
}

func mergeRouteMaps(rms1, rms2 []*types.RouteMap) ([]*types.RouteMap, error) {
	var rms []*types.RouteMap

	for _, rm := range append(rms1, rms2...) {
		if slices.ContainsFunc(rms, func(r *types.RouteMap) bool {
			return r.Name == rm.Name && r.Order == rm.Order
		}) {
			return nil, fmt.Errorf("duplicate route-map %q with equal order %d", rm.Name, rm.Order)
		}
		rms = append(rms, rm)
	}
	slices.SortFunc(rms, func(a, b *types.RouteMap) int {
		nameDiff := strings.Compare(a.Name, b.Name)
		if nameDiff != 0 {
			return nameDiff
		}
		return a.Order - b.Order
	})

	return rms, nil
}

func mergePrefixLists(pfs1, pfs2 []*types.IPPrefixList) []*types.IPPrefixList {
	var pfs []*types.IPPrefixList

	for _, pf := range append(pfs1, pfs2...) {
		if slices.ContainsFunc(pfs, func(p *types.IPPrefixList) bool {
			return cmp.Diff(p, pf) == ""
		}) {
			continue
		}
		pfs = append(pfs, pf)
	}

	slices.SortFunc(pfs, func(a, b *types.IPPrefixList) int {
		return strings.Compare(a.Name, b.Name)
	})

	return pfs
}

func interfaceSmallerFunc(a, b string) int {
	numsA := interfaceNumbers.FindAllString(a, -1)
	numsB := interfaceNumbers.FindAllString(b, -1)

	for i := range min(len(numsA), len(numsB)) {
		numA, _ := strconv.Atoi(numsA[i])
		numB, _ := strconv.Atoi(numsB[i])
		diff := numA - numB

		if diff != 0 {
			return diff
		}
	}

	if len(numsA) != len(numsB) {
		return len(numsA) - len(numsB)
	}

	return strings.Compare(a, b)
}

func isPortStaticNeighbor(port string, bgps []*types.BGP) bool {
	for _, bgp := range bgps {
		if slices.ContainsFunc(bgp.Neighbors, func(neigh types.BGPNeighbor) bool {
			return neigh.Name == fmt.Sprintf("%s interface", port)
		}) {
			return true
		}
	}
	return false
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

func addVRFtoFRRConfig(vrf *string, vrfs []string) []string {
	if vrf == nil {
		return vrfs
	}
	if pointer.SafeDeref(vrf) == "default" {
		return vrfs
	}
	if slices.Contains(vrfs, pointer.SafeDeref(vrf)) {
		return vrfs
	}
	return append(vrfs, pointer.SafeDeref(vrf))
}

func addVRF(vrf *string, vrfs []*types.Vrf) ([]*types.Vrf, error) {
	if vrf == nil {
		return vrfs, nil
	}
	if pointer.SafeDeref(vrf) == "default" {
		return vrfs, nil
	}

	vni, err := strconv.ParseUint(pointer.SafeDeref(vrf)[3:], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vni of vrf %s: %w", pointer.SafeDeref(vrf), err)
	}

	if !slices.ContainsFunc(vrfs, func(v *types.Vrf) bool {
		return v.VNI == uint32(vni)
	}) {
		vrfs = append(vrfs, &types.Vrf{
			Name: pointer.SafeDeref(vrf),
			VNI:  uint32(vni),
		})
	}

	return vrfs, nil
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

func addUnderlayBGPPort(intf *types.Interface, bgp *types.BGP, underlayPorts []string, routeMap string) {
	if bgp.PeerGroups == nil {
		// FIXME: use constants for peer group names
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
			GlobalCommands: []string{
				fmt.Sprintf("redistribute connected route-map %s", routeMap),
				fmt.Sprintf("redistribute kernel route-map %s", routeMap),
			},
			Neighbors: []*types.AddressFamilyNeighbor{
				{
					Name:     "FIREWALL",
					Commands: []string{"allowas-in 2"},
				},
			},
		}
	}

	if bgp.AddressFamilies.IPv6Unicast == nil {
		bgp.AddressFamilies.IPv6Unicast = &types.BGPAddressFamily{
			GlobalCommands: []string{
				fmt.Sprintf("redistribute connected route-map %s", routeMap),
				fmt.Sprintf("redistribute kernel route-map %s", routeMap),
			},
			Neighbors: []*types.AddressFamilyNeighbor{
				{
					Name:     "FIREWALL",
					Commands: []string{"activate", "allowas-in 2"},
				},
			},
		}
	}

	if bgp.AddressFamilies.L2VPNEVPN == nil {
		bgp.AddressFamilies.L2VPNEVPN = &types.BGPAddressFamily{
			GlobalCommands: []string{"advertise-all-vni"},
			Neighbors: []*types.AddressFamilyNeighbor{
				{
					Name:     "FABRIC",
					Commands: []string{"activate", "allowas-in 2"},
				},
				{
					Name:     "FIREWALL",
					Commands: []string{"activate", "allowas-in 2"},
				},
			},
		}
	}

	if slices.Contains(underlayPorts, intf.Name) {
		return
	}

	neighs := bgp.AddressFamilies.IPv4Unicast.Neighbors
	neighs = append(neighs, &types.AddressFamilyNeighbor{
		Name:     intf.Name,
		Commands: []string{fmt.Sprintf("route-map fw-%s-in in", intf.Name)},
	})
	bgp.AddressFamilies.IPv4Unicast.Neighbors = neighs

	neighs = bgp.AddressFamilies.IPv6Unicast.Neighbors
	neighs = append(neighs, &types.AddressFamilyNeighbor{
		Name:     intf.Name,
		Commands: []string{fmt.Sprintf("route-map fw-%s-in in", intf.Name)},
	})
	bgp.AddressFamilies.IPv6Unicast.Neighbors = neighs

	neighs = bgp.AddressFamilies.L2VPNEVPN.Neighbors
	neighs = append(neighs, &types.AddressFamilyNeighbor{
		Name:     intf.Name,
		Commands: []string{fmt.Sprintf("route-map fw-%s-vni out", intf.Name)},
	})
	bgp.AddressFamilies.L2VPNEVPN.Neighbors = neighs
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
			GlobalCommands: []string{
				"redistribute connected",
			},
			Neighbors: []*types.AddressFamilyNeighbor{
				{
					Name: "MACHINE",
					Commands: []string{
						"maximum-prefix 24000",
						fmt.Sprintf("route-map %s-in in", pointer.SafeDeref(intf.Vrf)),
					},
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
			GlobalCommands: []string{
				"redistribute connected",
			},
			Neighbors: []*types.AddressFamilyNeighbor{
				{
					Name: "MACHINE",
					Commands: []string{
						"activate",
						"maximum-prefix 24000",
						fmt.Sprintf("route-map %s-in6 in", pointer.SafeDeref(intf.Vrf)),
					},
				},
			},
		}
	}

	if bgp.AddressFamilies.L2VPNEVPN == nil {
		var globalCommands []string
		if has4 {
			globalCommands = append(globalCommands, "advertise ipv4 unicast")
		}
		if has6 {
			globalCommands = append(globalCommands, "advertise ipv6 unicast")
		}

		bgp.AddressFamilies.L2VPNEVPN = &types.BGPAddressFamily{
			GlobalCommands: globalCommands,
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
