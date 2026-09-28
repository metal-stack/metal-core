package redis

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const sep = "|"

var (
	// baseConfigDB is a leaf with two machine ports, a PXE VLAN, boot ACL tables, and a VTEP.
	baseConfigDB = test.StringMap{
		"ACL_TABLE": test.StringMap{
			"BOOT_V6": test.StringMap{
				"policy_desc": "MEP-20 boot vrf ingress filter",
				"stage":       "ingress",
				"type":        "L3V6",
			},
			"BOOT_V4": test.StringMap{
				"policy_desc": "MEP-20 boot vrf ingress filter",
				"stage":       "ingress",
				"type":        "L3",
			},
			"ALLOW_SSH": test.StringMap{
				"stage": "ingress",
				"type":  "CTRLPLANE",
			},
		},
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{"admin_status": "up", "alias": "Eth1/1", "index": "1", "lanes": "1,2,3,4", "mtu": "9000"},
			"Ethernet4": test.StringMap{"admin_status": "up", "alias": "Eth2/1", "index": "2", "lanes": "5,6,7,8", "mtu": "9000"},
		},
		"VLAN": test.StringMap{
			"Vlan4000": test.StringMap{"vlanid": "4000"},
		},
		"VXLAN_TUNNEL": test.StringMap{
			"vtep": test.StringMap{"src_ip": "10.0.0.1"},
		},
	}

	bootConf = &types.BootConf{
		Vrf:    types.BootVrfName,
		VNI:    104000,
		VLANID: 1001,
		RDNSS:  []string{"fd00:20:ffff::53"},
		Ports: map[string]types.BootPort{
			"Ethernet0": {Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"},
			"Ethernet4": {Prefix: "fd00:20:0:101::/64", Address: "fd00:20:0:101::1/64"},
		},
	}
)

type testApplier struct {
	*Applier
	asicClient   *db.Client
	configClient valkey.Client
	config       test.HashMap
}

func newTestApplier(t *testing.T, data test.StringMap) *testApplier {
	t.Helper()
	ctx := t.Context()

	appl := test.StartValkey(t)
	asic := test.StartValkey(t)
	config := test.StartValkey(t)
	counters := test.StartValkey(t)
	t.Cleanup(func() {
		appl.Close()
		asic.Close()
		config.Close()
		counters.Close()
	})

	require.NoError(t, test.LoadData(ctx, config, data, sep))

	a := NewApplier(slog.Default(), db.NewWithClients(appl, asic, config, counters, sep))
	// force the applier to apply the first configuration
	a.previousCfg = &types.Conf{Name: "previous"}

	return &testApplier{
		Applier:      a,
		asicClient:   db.NewClient(asic, sep),
		configClient: config,
	}
}

func (a *testApplier) apply(t *testing.T, cfg *types.Conf) {
	t.Helper()
	require.NoError(t, a.Apply(t.Context(), cfg))
	a.refresh(t)
}

func (a *testApplier) refresh(t *testing.T) {
	t.Helper()
	data, err := test.GetData(t.Context(), a.configClient, sep)
	require.NoError(t, err)
	a.config = data
}

func (a *testApplier) keysWithPrefix(prefix string) []string {
	var keys []string
	for k := range a.config {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys
}

func pxeConf(unprovisioned []string) *types.Conf {
	return &types.Conf{
		Name:      "leaf01",
		PXEVlanID: 4000,
		Ports: types.Ports{
			Unprovisioned: unprovisioned,
			Vrfs:          map[string]*types.Vrf{},
			Firewalls:     map[string]*types.Firewall{},
			AdminStatus:   map[string]types.PortStatus{},
		},
	}
}

func l3Conf(unprovisioned []string) *types.Conf {
	c := pxeConf(unprovisioned)
	boot := *bootConf
	c.Boot = &boot
	return c
}

func TestApplier_PxeToL3(t *testing.T) {
	ctx := context.Background()
	a := newTestApplier(t, baseConfigDB)

	// start in PXE mode
	a.apply(t, pxeConf([]string{"Ethernet0", "Ethernet4"}))
	require.Equal(t, map[string]string{"tagging_mode": "untagged"}, a.config["VLAN_MEMBER|Vlan4000|Ethernet0"])
	require.Equal(t, map[string]string{"tagging_mode": "untagged"}, a.config["VLAN_MEMBER|Vlan4000|Ethernet4"])
	require.Empty(t, a.keysWithPrefix("INTERFACE|"))
	require.Empty(t, a.config["ACL_TABLE|BOOT_V6"]["ports@"])

	// migrate to L3 boot
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	require.Empty(t, a.keysWithPrefix("VLAN_MEMBER|"), "ports must be removed from the pxe vlan")
	require.Equal(t, map[string]string{"fallback": "false", "vni": "104000"}, a.config["VRF|VrfBoot"])
	require.Equal(t, map[string]string{"vlanid": "1001"}, a.config["VLAN|Vlan1001"])
	require.Equal(t, map[string]string{"suppress": "on"}, a.config["SUPPRESS_VLAN_NEIGH|Vlan1001"])
	require.Equal(t, map[string]string{"vrf_name": "VrfBoot"}, a.config["VLAN_INTERFACE|Vlan1001"])
	require.Equal(t, map[string]string{"vlan": "Vlan1001", "vni": "104000"}, a.config["VXLAN_TUNNEL_MAP|vtep|map_104000_Vlan1001"])

	for port, bp := range bootConf.Ports {
		require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "VrfBoot"}, a.config["INTERFACE|"+port], port)
		require.Equal(t, map[string]string{"NULL": "NULL"}, a.config["INTERFACE|"+port+"|"+bp.Address], port)
		require.Equal(t, "9000", a.config["PORT|"+port]["mtu"])
	}

	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V4"]["ports@"])
	require.Empty(t, a.config["ACL_TABLE|ALLOW_SSH"]["ports@"], "non boot acls must not be touched")

	// applying the same config again must be a noop
	before := a.config
	require.NoError(t, a.Apply(ctx, l3Conf([]string{"Ethernet0", "Ethernet4"})))
	a.refresh(t)
	require.Equal(t, before, a.config)
}

func TestApplier_L3ToTenantAndBack(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	// Ethernet0 gets allocated
	c := l3Conf([]string{"Ethernet4"})
	c.Ports.Vrfs["Vrf104001"] = &types.Vrf{VNI: 104001, VLANID: 1002, Neighbors: []string{"Ethernet0"}}
	a.apply(t, c)

	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf104001"}, a.config["INTERFACE|Ethernet0"])
	require.Equal(t, []string{"INTERFACE|Ethernet0"}, a.keysWithPrefix("INTERFACE|Ethernet0"), "boot address must be removed")
	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "VrfBoot"}, a.config["INTERFACE|Ethernet4"])
	require.Equal(t, "Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
	require.Equal(t, map[string]string{"fallback": "false", "vni": "104000"}, a.config["VRF|VrfBoot"], "boot vrf must survive")
	require.Equal(t, map[string]string{"fallback": "false", "vni": "104001"}, a.config["VRF|Vrf104001"])

	// Ethernet0 gets freed again
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "VrfBoot"}, a.config["INTERFACE|Ethernet0"])
	require.Equal(t, map[string]string{"NULL": "NULL"}, a.config["INTERFACE|Ethernet0|fd00:20:0:100::1/64"])
	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
	require.Empty(t, a.config["VRF|Vrf104001"], "unused tenant vrf must be cleaned up")
	require.Equal(t, map[string]string{"fallback": "false", "vni": "104000"}, a.config["VRF|VrfBoot"])
}

func TestApplier_L3ToFirewall(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	c := l3Conf([]string{"Ethernet4"})
	c.Ports.Firewalls["Ethernet0"] = &types.Firewall{Port: "Ethernet0"}
	a.apply(t, c)

	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable"}, a.config["INTERFACE|Ethernet0"], "firewall port must be in the default vrf")
	require.Equal(t, []string{"INTERFACE|Ethernet0"}, a.keysWithPrefix("INTERFACE|Ethernet0"))
	require.Equal(t, "9216", a.config["PORT|Ethernet0"]["mtu"])
	require.Equal(t, "Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
}

func TestApplier_L3ToPxeRollback(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	a.apply(t, pxeConf([]string{"Ethernet0", "Ethernet4"}))

	require.Empty(t, a.keysWithPrefix("INTERFACE|"), "boot interfaces must be removed")
	require.Equal(t, map[string]string{"tagging_mode": "untagged"}, a.config["VLAN_MEMBER|Vlan4000|Ethernet0"])
	require.Equal(t, map[string]string{"tagging_mode": "untagged"}, a.config["VLAN_MEMBER|Vlan4000|Ethernet4"])
	require.Empty(t, a.config["VRF|VrfBoot"], "boot vrf must be removed")
	require.Empty(t, a.config["VLAN|Vlan1001"])
	require.Empty(t, a.config["VLAN_INTERFACE|Vlan1001"])
	require.Empty(t, a.config["SUPPRESS_VLAN_NEIGH|Vlan1001"])
	require.Empty(t, a.config["VXLAN_TUNNEL_MAP|vtep|map_104000_Vlan1001"])
	require.Equal(t, "", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
	require.Equal(t, "", a.config["ACL_TABLE|BOOT_V4"]["ports@"])
	require.Equal(t, map[string]string{"vlanid": "4000"}, a.config["VLAN|Vlan4000"], "pxe vlan must survive")
}

func TestApplier_L3RequiresBootACLTables(t *testing.T) {
	data := test.StringMap{}
	for k, v := range baseConfigDB {
		if k != "ACL_TABLE" {
			data[k] = v
		}
	}
	a := newTestApplier(t, data)

	err := a.Apply(t.Context(), l3Conf([]string{"Ethernet0", "Ethernet4"}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no acl tables with prefix BOOT_")

	// nothing must be touched, especially no port may be routed in the boot VRF without the filter
	a.refresh(t)
	require.Empty(t, a.keysWithPrefix("INTERFACE|"))
	require.Empty(t, a.config["VRF|VrfBoot"])

	// PXE mode does not need the ACL tables
	require.NoError(t, a.Apply(t.Context(), pxeConf([]string{"Ethernet0", "Ethernet4"})))
}

func TestApplier_L3MissingBootPrefixIsAnError(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)

	c := l3Conf([]string{"Ethernet0", "Ethernet4"})
	c.Boot.Ports = map[string]types.BootPort{"Ethernet0": bootConf.Ports["Ethernet0"]}
	err := a.Apply(t.Context(), c)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no boot prefix for unprovisioned port Ethernet4")
}

func TestApplier_ACLBindingSurvivesFailedTransition(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	// Ethernet0 gets allocated, but the transition fails because the port vanished from the PORT table
	require.NoError(t, a.db.Config.Client().Del(t.Context(), db.Key{"PORT", "Ethernet0"}))
	c := l3Conf([]string{"Ethernet4"})
	c.Ports.Vrfs["Vrf104001"] = &types.Vrf{VNI: 104001, VLANID: 1002, Neighbors: []string{"Ethernet0"}}
	err := a.Apply(t.Context(), c)
	require.ErrorContains(t, err, "port Ethernet0 does not exist in CONFIG_DB")
	a.refresh(t)

	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"], "a port with a failed transition must stay filtered")
	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V4"]["ports@"])

	// once the transition succeeds, the binding is removed
	require.NoError(t, a.db.Config.Client().HSet(t.Context(), db.Key{"PORT", "Ethernet0"}, db.Val{"admin_status": "up", "mtu": "9000"}))
	a.apply(t, c)
	require.Equal(t, "Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
}

func TestApplier_PxeRollbackKeepsBindingOnFailure(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	require.NoError(t, a.db.Config.Client().Del(t.Context(), db.Key{"PORT", "Ethernet4"}))
	err := a.Apply(t.Context(), pxeConf([]string{"Ethernet0", "Ethernet4"}))
	require.ErrorContains(t, err, "port Ethernet4 does not exist in CONFIG_DB")
	a.refresh(t)

	require.Equal(t, "Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"], "only the port that left the boot vrf is unbound")
}

func TestApplier_FailedBootTransitionDoesNotDuplicateBinding(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	// Ethernet4 is bound already and now fails its boot transition because its prefix is missing
	c := l3Conf([]string{"Ethernet0", "Ethernet4"})
	c.Boot.Ports = map[string]types.BootPort{"Ethernet0": bootConf.Ports["Ethernet0"]}
	err := a.Apply(t.Context(), c)
	require.ErrorContains(t, err, "no boot prefix for unprovisioned port Ethernet4")
	a.refresh(t)

	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V6"]["ports@"])
	require.Equal(t, "Ethernet0,Ethernet4", a.config["ACL_TABLE|BOOT_V4"]["ports@"])
	require.Equal(t, []string{"Ethernet0", "Ethernet4"}, c.Ports.Unprovisioned, "the config must not be modified")
}

func TestApplier_MoveRefreshesRifOidMap(t *testing.T) {
	a := newTestApplier(t, baseConfigDB)
	a.apply(t, l3Conf([]string{"Ethernet0", "Ethernet4"}))

	// the RIF shows up in COUNTERS_DB and ASIC_DB only after the initial OID map refresh
	ctx := t.Context()
	require.NoError(t, a.db.Counters.Client().HSet(ctx, db.Key{"COUNTERS_RIF_NAME_MAP"}, db.Val{"Ethernet0": "oid:0x6000000000001"}))
	require.NoError(t, a.asicClient.HSet(ctx, db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", "oid:0x6000000000001"}, db.Val{"SAI_ROUTER_INTERFACE_ATTR_TYPE": "SAI_ROUTER_INTERFACE_TYPE_PORT"}))
	a.previousCfg = &types.Conf{Name: "force"}
	a.rifOidMap = map[string]db.OID{}

	// the move must wait for the RIF to disappear; simulate the ASIC releasing it
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = a.asicClient.Del(context.Background(), db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", "oid:0x6000000000001"})
	}()

	c := l3Conf([]string{"Ethernet4"})
	c.Ports.Vrfs["Vrf104001"] = &types.Vrf{VNI: 104001, VLANID: 1002, Neighbors: []string{"Ethernet0"}}
	a.apply(t, c)

	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf104001"}, a.config["INTERFACE|Ethernet0"])
	require.Equal(t, db.OID("oid:0x6000000000001"), a.rifOidMap["Ethernet0"], "rif oid map must have been refreshed")
}
