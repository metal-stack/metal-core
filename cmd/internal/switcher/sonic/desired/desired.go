// Package desired turns the configuration of a switch, which assigns each port its role, into the CONFIG_DB
// entries that configure SONiC for it.
package desired

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/vrfentries"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	// mtuMachine is the mtu of ports with machines and of unprovisioned ports.
	mtuMachine = "9000"
	// mtuFabric is the mtu of the ports to the spines and to firewalls.
	mtuFabric = "9216"
)

// Switch is the configuration of the switch as CONFIG_DB entries.
type Switch struct {
	Objects []object.Object
	// Vrfs are the entries of each tenant vrf, by vrf name.
	Vrfs map[string][]object.Object
	// RouterInterfaces are the router interfaces of the ports of the tenant vrfs, by port.
	RouterInterfaces map[string]object.RouterInterface
	// Ports are the ports the configuration assigns a role, sorted.
	Ports []string
}

// Build returns the CONFIG_DB entries for the configuration:
//   - an underlay port to a spine and a firewall port are router interfaces in the default vrf,
//   - an unprovisioned port is a member of the PXE vlan,
//   - a port of a machine is a router interface in the vrf of the machine,
//   - a tenant vrf consists of the entries the VrfEntries quirk returns.
func Build(cfg *types.Conf, vrfEntries vrfentries.VrfEntries, vtep string) (*Switch, error) {
	b := &builder{
		sw: &Switch{
			Vrfs:             map[string][]object.Object{},
			RouterInterfaces: map[string]object.RouterInterface{},
		},
		keys:        map[string]bool{},
		adminStatus: cfg.Ports.AdminStatus,
	}

	for _, port := range cfg.Ports.Underlay {
		b.port(port, mtuFabric, object.NewRouterInterface(port, ""))
	}

	pxeVlan := object.VlanName(cfg.PXEVlanID)
	for _, port := range cfg.Ports.Unprovisioned {
		b.port(port, mtuMachine, object.NewVlanMember(pxeVlan, port))
	}

	for _, name := range slices.Sorted(maps.Keys(cfg.Ports.Firewalls)) {
		fw := cfg.Ports.Firewalls[name]
		b.port(fw.Port, mtuFabric, object.NewRouterInterface(fw.Port, ""))
	}

	for _, name := range slices.Sorted(maps.Keys(cfg.Ports.Vrfs)) {
		vrf := cfg.Ports.Vrfs[name]

		objects := vrfEntries.Objects(name, vrf, vtep)
		b.sw.Vrfs[name] = objects
		b.add(objects...)

		for _, port := range vrf.Neighbors {
			ri := object.NewRouterInterface(port, name)
			b.sw.RouterInterfaces[port] = ri
			b.port(port, mtuMachine, ri)
		}
	}

	if len(b.duplicates) > 0 {
		return nil, fmt.Errorf("the configuration has these entries more than once: %s", strings.Join(b.duplicates, ", "))
	}
	slices.Sort(b.sw.Ports)
	return b.sw, nil
}

type builder struct {
	sw          *Switch
	keys        map[string]bool
	duplicates  []string
	adminStatus map[string]types.PortStatus
}

// port adds the port with its mtu, its admin status and the entry of its role.
func (b *builder) port(name, mtu string, role object.Object) {
	b.sw.Ports = append(b.sw.Ports, name)
	b.add(object.NewPort(name, mtu, b.adminStatus[name]), role)
}

// add adds the objects. A port with two roles shows as a duplicate.
func (b *builder) add(objects ...object.Object) {
	for _, o := range objects {
		key := strings.Join(o.Key(), "|")
		if b.keys[key] {
			b.duplicates = append(b.duplicates, key)
			continue
		}
		b.keys[key] = true
		b.sw.Objects = append(b.sw.Objects, o)
	}
}
