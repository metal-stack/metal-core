package sonictest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
)

// Daemons apply the CONFIG_DB entries metal-core manages like the SONiC daemons do, one Sync at a time:
//   - vrfmgrd, vlanmgrd, intfmgrd and vxlanmgrd report what they set up in the STATE_DB and APPL_DB,
//   - orchagent creates virtual routers, router interfaces and bridge ports in the ASIC_DB. Like the real one, it
//     keeps a router interface as long as a route uses it, and a virtual router as long as a router interface or
//     a route references it,
//   - the kernel takes the admin status of the ports.
//
// Routes are not derived, tests set them in the APPL_DB ROUTE_TABLE.
type Daemons struct {
	// Stuck are CONFIG_DB keys whose changes the daemons do not apply, like "INTERFACE|Ethernet1".
	Stuck map[string]bool
	// Kernel holds the admin status of the ports.
	Kernel Kernel

	d *Databases
	// derived are the entries each CONFIG_DB entry was applied as, by CONFIG_DB key
	derived map[string][]Entry
	// written are the entries the last Sync wrote, by database and key
	written map[string]Entry
	// rifVrf is the vrf of the router interface of a port
	rifVrf map[string]string
}

// Entry is an entry of a database, keyed like in the database.
type Entry struct {
	DB     string
	Key    string
	Fields map[string]string
}

const (
	asicRif        = "ASIC_STATE:SAI_OBJECT_TYPE_ROUTER_INTERFACE:"
	asicBridgePort = "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:"
	vrfObjectTable = "VRF_OBJECT_TABLE|"
)

// Daemons returns the daemons of the databases.
func (d *Databases) Daemons() *Daemons {
	return &Daemons{
		Stuck:   map[string]bool{},
		Kernel:  Kernel{},
		d:       d,
		derived: map[string][]Entry{},
		written: map[string]Entry{},
		rifVrf:  map[string]string{},
	}
}

// Sync applies the CONFIG_DB.
func (s *Daemons) Sync(t testing.TB) {
	t.Helper()

	config := s.d.Get(t, "CONFIG_DB")
	appl := s.d.Get(t, "APPL_DB")
	counters := s.d.Get(t, "COUNTERS_DB")

	derived := map[string][]Entry{}
	for _, key := range slices.Sorted(maps.Keys(config)) {
		if s.Stuck[key] {
			continue
		}
		derived[key] = s.derive(key, config[key], config, counters)
	}
	for key, entries := range s.derived {
		if s.Stuck[key] {
			derived[key] = entries
		}
	}

	want := map[string]Entry{}
	for _, entries := range derived {
		for _, e := range entries {
			id := e.DB + " " + e.Key
			// the COUNTERS_DB maps hold the oids of all ports in one entry
			if prev, ok := want[id]; ok {
				e.Fields = maps.Clone(e.Fields)
				maps.Copy(e.Fields, prev.Fields)
			}
			want[id] = e
		}
	}

	// orchagent keeps what is still referenced
	routes := routes(appl)
	for id, e := range s.written {
		if _, ok := want[id]; ok {
			continue
		}
		switch {
		case e.DB == "ASIC_DB" && strings.HasPrefix(e.Key, asicRif):
			port := e.Fields["port"]
			if routes.via(s.rifVrf[port], port) {
				want[id] = e
			}
		case e.DB == "STATE_DB" && strings.HasPrefix(e.Key, vrfObjectTable):
			vrf := strings.TrimPrefix(e.Key, vrfObjectTable)
			if routes.in(vrf) || s.rifIn(want, vrf) {
				want[id] = e
			}
		}
	}

	s.write(t, want)
	s.derived = derived
	s.written = want
}

// derive returns the entries the daemons apply the CONFIG_DB entry as.
func (s *Daemons) derive(key string, fields map[string]string, config, counters test.HashMap) []Entry {
	parts := strings.Split(key, "|")
	table, name := parts[0], ""
	if len(parts) > 1 {
		name = parts[1]
	}

	switch {
	case table == "VRF" && len(parts) == 2:
		entries := []Entry{
			{DB: "STATE_DB", Key: "VRF_TABLE|" + name, Fields: map[string]string{"state": "ok"}},
			{DB: "STATE_DB", Key: vrfObjectTable + name, Fields: map[string]string{"state": "ok"}},
			{DB: "APPL_DB", Key: "VRF_TABLE:" + name, Fields: without(fields, "NULL")},
		}
		if vni := fields["vni"]; vni != "" {
			entries = append(entries, Entry{DB: "APPL_DB", Key: "VXLAN_VRF_TABLE:vtep:evpn_map_" + vni + "_" + name, Fields: map[string]string{"vni": vni, "vrf": name}})
		}
		return entries

	case table == "VLAN" && len(parts) == 2:
		return []Entry{{DB: "STATE_DB", Key: "VLAN_TABLE|" + name, Fields: map[string]string{"state": "ok"}}}

	case table == "SUPPRESS_VLAN_NEIGH" && len(parts) == 2:
		return []Entry{{DB: "STATE_DB", Key: "SUPPRESS_VLAN_NEIGH_TABLE|" + name, Fields: map[string]string{"netdev": "vtep-" + name}}}

	case table == "VLAN_INTERFACE" && len(parts) == 2:
		vrf := fields["vrf_name"]
		if vrf != "" && config["VRF|"+vrf] == nil {
			// intfmgrd waits for the vrf
			return nil
		}
		return []Entry{
			{DB: "STATE_DB", Key: "INTERFACE_TABLE|" + name, Fields: map[string]string{"vrf": vrf}},
			{DB: "APPL_DB", Key: "INTF_TABLE:" + name, Fields: map[string]string{"vrf_name": vrf}},
		}

	case table == "VXLAN_TUNNEL_MAP" && len(parts) == 3:
		return []Entry{{DB: "APPL_DB", Key: "VXLAN_TUNNEL_MAP_TABLE:" + parts[1] + ":" + parts[2], Fields: without(fields, "NULL")}}

	case table == "INTERFACE" && len(parts) == 2 && strings.HasPrefix(name, "Ethernet"):
		vrf := fields["vrf_name"]
		if vrf != "" && config["VRF|"+vrf] == nil {
			// intfmgrd waits for the vrf
			return nil
		}
		s.rifVrf[name] = vrf
		return []Entry{
			{DB: "STATE_DB", Key: "INTERFACE_TABLE|" + name, Fields: map[string]string{"vrf": vrf}},
			{DB: "COUNTERS_DB", Key: "COUNTERS_RIF_NAME_MAP", Fields: map[string]string{name: s.oid(counters, "COUNTERS_RIF_NAME_MAP", name, "0x60000000")}},
			{DB: "ASIC_DB", Key: asicRif + s.oid(counters, "COUNTERS_RIF_NAME_MAP", name, "0x60000000"), Fields: map[string]string{"port": name}},
		}

	case table == "VLAN_MEMBER" && len(parts) == 3:
		port := parts[2]
		return []Entry{
			{DB: "STATE_DB", Key: "VLAN_MEMBER_TABLE|" + parts[1] + "|" + port, Fields: map[string]string{"state": "ok"}},
			{DB: "ASIC_DB", Key: asicBridgePort + "oid:0x3a0000000" + strings.TrimPrefix(port, "Ethernet"), Fields: map[string]string{
				"SAI_BRIDGE_PORT_ATTR_PORT_ID": s.oid(counters, "COUNTERS_PORT_NAME_MAP", port, "0x10000000"),
			}},
		}

	case table == "PORT" && len(parts) == 2:
		if status := fields["admin_status"]; status != "" {
			s.Kernel[name] = status == "up"
		}
		return []Entry{{DB: "COUNTERS_DB", Key: "COUNTERS_PORT_NAME_MAP", Fields: map[string]string{name: s.oid(counters, "COUNTERS_PORT_NAME_MAP", name, "0x10000000")}}}
	}
	return nil
}

// oid returns the oid of the name in the COUNTERS_DB map, or a new one.
func (s *Daemons) oid(counters test.HashMap, table, name, prefix string) string {
	if oid := counters[table][name]; oid != "" {
		return oid
	}
	return fmt.Sprintf("oid:%s%s", prefix, strings.TrimPrefix(name, "Ethernet"))
}

// rifIn tells whether a router interface of a port in the vrf is wanted.
func (s *Daemons) rifIn(want map[string]Entry, vrf string) bool {
	for _, e := range want {
		if e.DB == "ASIC_DB" && strings.HasPrefix(e.Key, asicRif) && s.rifVrf[e.Fields["port"]] == vrf {
			return true
		}
	}
	return false
}

// write makes the tables the daemons write hold exactly the wanted entries. The COUNTERS_DB maps are merged.
func (s *Daemons) write(t testing.TB, want map[string]Entry) {
	t.Helper()

	for id, e := range s.written {
		if e.DB == "COUNTERS_DB" {
			continue
		}
		if _, ok := want[id]; !ok {
			vc, _ := s.d.Client(t, e.DB)
			require.NoError(t, vc.Do(t.Context(), vc.B().Del().Key(e.Key).Build()).Error())
		}
	}
	for _, id := range slices.Sorted(maps.Keys(want)) {
		e := want[id]
		vc, _ := s.d.Client(t, e.DB)
		cmd := vc.B().Hset().Key(e.Key).FieldValue()
		fields := e.Fields
		if len(fields) == 0 {
			fields = map[string]string{"NULL": "NULL"}
		}
		for f, v := range fields {
			cmd = cmd.FieldValue(f, v)
		}
		require.NoError(t, vc.Do(t.Context(), cmd.Build()).Error())
	}
}

func without(fields map[string]string, field string) map[string]string {
	m := maps.Clone(fields)
	delete(m, field)
	return m
}

// routeList are the routes of the APPL_DB by key, with the interfaces of their next hops.
type routeList map[string][]string

func routes(appl test.HashMap) routeList {
	l := routeList{}
	for key, fields := range appl {
		if strings.HasPrefix(key, "ROUTE_TABLE:") {
			l[key] = strings.Split(fields["ifname"], ",")
		}
	}
	return l
}

// in tells whether there are routes in the vrf that orchagent programs, i.e. no IPv6 link-local ones.
func (l routeList) in(vrf string) bool {
	for key := range l {
		if strings.HasPrefix(key, "ROUTE_TABLE:"+vrf+":") && !strings.HasPrefix(key, "ROUTE_TABLE:"+vrf+":fe80:") {
			return true
		}
	}
	return false
}

// via tells whether there are programmed routes via the port in the vrf. The default vrf is "".
func (l routeList) via(vrf, port string) bool {
	for key, ifnames := range l {
		rest := strings.TrimPrefix(key, "ROUTE_TABLE:")
		if vrf != "" {
			var ok bool
			if rest, ok = strings.CutPrefix(rest, vrf+":"); !ok {
				continue
			}
		} else if strings.HasPrefix(rest, "Vrf") {
			continue
		}
		if !strings.HasPrefix(rest, "fe80:") && slices.Contains(ifnames, port) {
			return true
		}
	}
	return false
}
