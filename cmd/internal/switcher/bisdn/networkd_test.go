package bisdn

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type fakeLinks struct {
	deleted  []string
	released []string
}

func (f *fakeLinks) Delete(name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func (f *fakeLinks) Release(name string) error {
	f.released = append(f.released, name)
	return nil
}

func (f *fakeLinks) reset() {
	f.deleted, f.released = nil, nil
}

func testConf() *types.Conf {
	cfg := &types.Conf{
		Name:          "leaf01",
		Loopback:      "10.0.0.10",
		ASN:           4200000010,
		MetalCoreCIDR: "10.255.255.2/24",
		PXEVlanID:     4000,
		Ports: types.Ports{
			Underlay:      []string{"port1"},
			Unprovisioned: []string{"port4"},
			Vrfs: types.Vrfs{
				"vrf104001": {VNI: 104001, VLANID: 1001, Neighbors: []string{"port2"}},
				"vrf104002": {VNI: 104002, VLANID: 1002, Neighbors: []string{"port3"}},
			},
		},
	}
	cfg.CapitalizeVrfName()
	return cfg
}

func newTestApplier(t *testing.T) (*networkdApplier, *fakeLinks, *int) {
	reloads := 0
	links := &fakeLinks{}
	return &networkdApplier{
		dir:   t.TempDir(),
		tpl:   templates.BISDNNetworkdTemplates(""),
		links: links,
		reload: func(context.Context) error {
			reloads++
			return nil
		},
		log: slog.Default(),
	}, links, &reloads
}

func managedNames(t *testing.T, dir string) []string {
	files, err := readManagedFiles(dir)
	require.NoError(t, err)
	return slices.Sorted(maps.Keys(files))
}

func TestNetworkdApply(t *testing.T) {
	ctx := context.Background()
	a, links, reloads := newTestApplier(t)

	// files metal-core doesn't own: metal-roles' mgmt VRF and management
	// port, other files and look-alikes
	foreign := map[string]string{
		"10-mgmt.netdev":               "[NetDev]\nName=mgmt\nKind=vrf\n",
		"10-mgmt.network":              "[Match]\nName=mgmt\n",
		"10-enp0.network":              "[Match]\nName=enp0\n\n[Network]\nVRF=mgmt\n",
		"50-nostest-x.network":         "[Match]\nName=port9\n",
		"42-vni104001.netdev":          "[NetDev]\nName=vni104001\n",
		"30-bisdn-Vrf104001.netdev":    "[NetDev]\nName=Vrf104001\n",
		"20-metal-core-port1.conf":     "x\n",
		"50-metal-core-port2.network~": "x\n",
	}
	for name, content := range foreign {
		require.NoError(t, os.WriteFile(filepath.Join(a.dir, name), []byte(content), 0o644))
	}
	defer func() {
		for name, content := range foreign {
			got, err := os.ReadFile(filepath.Join(a.dir, name))
			require.NoError(t, err, "files that metal-core doesn't own must stay")
			require.Equal(t, content, string(got), name)
		}
	}()

	require.NoError(t, a.Apply(ctx, testConf()))
	require.Equal(t, 1, *reloads)
	require.Empty(t, links.deleted)
	require.Empty(t, links.released)
	require.Equal(t, []string{
		"10-metal-core-lo.network",
		"20-metal-core-port1.network",
		"30-metal-core-Vrf104001.netdev", "30-metal-core-Vrf104001.network",
		"30-metal-core-Vrf104002.netdev", "30-metal-core-Vrf104002.network",
		"40-metal-core-bridge.netdev", "40-metal-core-bridge.network",
		"41-metal-core-vlan1001.netdev", "41-metal-core-vlan1001.network",
		"41-metal-core-vlan1002.netdev", "41-metal-core-vlan1002.network",
		"41-metal-core-vlan4000.netdev", "41-metal-core-vlan4000.network",
		"42-metal-core-vni104000.netdev", "42-metal-core-vni104000.network",
		"42-metal-core-vni104001.netdev", "42-metal-core-vni104001.network",
		"42-metal-core-vni104002.netdev", "42-metal-core-vni104002.network",
		"50-metal-core-port2.network",
		"50-metal-core-port3.network",
		"51-metal-core-port4.network",
	}, managedNames(t, a.dir))

	// unchanged config: no reload and no file is written again
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	for _, name := range managedNames(t, a.dir) {
		require.NoError(t, os.Chtimes(filepath.Join(a.dir, name), old, old))
	}
	require.NoError(t, a.Apply(ctx, testConf()))
	require.Equal(t, 1, *reloads)

	// the machine on port2 is freed and its tenant VRF goes away: all of the
	// VRF's files and netdevs go, port2 moves to the PXE VLAN
	cfg := testConf()
	delete(cfg.Ports.Vrfs, "Vrf104001")
	cfg.Ports.Unprovisioned = append(cfg.Ports.Unprovisioned, "port2")
	require.NoError(t, a.Apply(ctx, cfg))
	require.Equal(t, 2, *reloads)
	slices.Sort(links.deleted)
	require.Equal(t, []string{"Vrf104001", "vlan1001", "vni104001"}, links.deleted)
	require.Empty(t, links.released, "port2 still has a .network file")
	names := managedNames(t, a.dir)
	for _, gone := range []string{
		"30-metal-core-Vrf104001.netdev", "30-metal-core-Vrf104001.network",
		"41-metal-core-vlan1001.netdev", "41-metal-core-vlan1001.network",
		"42-metal-core-vni104001.netdev", "42-metal-core-vni104001.network",
		"50-metal-core-port2.network",
	} {
		require.NotContains(t, names, gone)
	}
	require.Contains(t, names, "51-metal-core-port2.network")
	require.Contains(t, names, "42-metal-core-vni104002.netdev", "other VRFs stay")
	for _, name := range names {
		info, err := os.Stat(filepath.Join(a.dir, name))
		require.NoError(t, err)
		switch name {
		case "40-metal-core-bridge.network", "51-metal-core-port2.network":
			require.True(t, info.ModTime().After(old), "%s must be written", name)
		default:
			require.True(t, info.ModTime().Equal(old), "%s is unchanged and must not be written", name)
		}
	}

	// port4 is no longer configured at all
	links.reset()
	cfg.Ports.Unprovisioned = []string{"port2"}
	require.NoError(t, a.Apply(ctx, cfg))
	require.Empty(t, links.deleted)
	require.Equal(t, []string{"port4"}, links.released)
	require.NotContains(t, managedNames(t, a.dir), "51-metal-core-port4.network")

	// port3 goes admin down and up again, staying in its VRF
	links.reset()
	cfg.Ports.AdminStatus = map[string]types.PortStatus{"port3": types.PortStatusDown}
	require.NoError(t, a.Apply(ctx, cfg))
	content, err := os.ReadFile(filepath.Join(a.dir, "50-metal-core-port3.network"))
	require.NoError(t, err)
	require.Contains(t, string(content), "ActivationPolicy=always-down\n")
	require.Contains(t, string(content), "VRF=Vrf104002\n")
	cfg.Ports.AdminStatus = nil
	require.NoError(t, a.Apply(ctx, cfg))
	content, err = os.ReadFile(filepath.Join(a.dir, "50-metal-core-port3.network"))
	require.NoError(t, err)
	require.NotContains(t, string(content), "ActivationPolicy")
	require.Empty(t, links.deleted)
	require.Empty(t, links.released)

	// a new loopback changes the VXLAN source, so the VTEPs are created again
	cfg.Loopback = "10.0.0.11"
	require.NoError(t, a.Apply(ctx, cfg))
	slices.Sort(links.deleted)
	require.Equal(t, []string{"vni104000", "vni104002"}, links.deleted)
	require.Empty(t, links.released)

	// the last VRF goes away, too
	links.reset()
	cfg.Ports.Vrfs = types.Vrfs{}
	cfg.Ports.Unprovisioned = []string{"port2", "port3"}
	require.NoError(t, a.Apply(ctx, cfg))
	slices.Sort(links.deleted)
	require.Equal(t, []string{"Vrf104002", "vlan1002", "vni104002"}, links.deleted)
	require.Empty(t, links.released)
	require.Equal(t, []string{
		"10-metal-core-lo.network",
		"20-metal-core-port1.network",
		"40-metal-core-bridge.netdev", "40-metal-core-bridge.network",
		"41-metal-core-vlan4000.netdev", "41-metal-core-vlan4000.network",
		"42-metal-core-vni104000.netdev", "42-metal-core-vni104000.network",
		"51-metal-core-port2.network",
		"51-metal-core-port3.network",
	}, managedNames(t, a.dir))
}

func TestNetworkdApplyRetriesReload(t *testing.T) {
	ctx := context.Background()
	a, _, reloads := newTestApplier(t)

	failing := true
	a.reload = func(context.Context) error {
		*reloads++
		if failing {
			return errors.New("networkd is not running")
		}
		return nil
	}

	require.Error(t, a.Apply(ctx, testConf()))
	failing = false
	require.NoError(t, a.Apply(ctx, testConf()))
	require.Equal(t, 2, *reloads, "unchanged files must be reloaded again after a failed reload")
	require.NoError(t, a.Apply(ctx, testConf()))
	require.Equal(t, 2, *reloads)
}

func TestRenderRejectsBadConfig(t *testing.T) {
	tpl := templates.BISDNNetworkdTemplates("")

	cfg := testConf()
	cfg.Ports.Unprovisioned = append(cfg.Ports.Unprovisioned, "port2")
	_, err := render(tpl, cfg)
	require.ErrorContains(t, err, "interface port2 is configured twice")

	cfg = testConf()
	cfg.Ports.Underlay = []string{"../port1"}
	_, err = render(tpl, cfg)
	require.ErrorContains(t, err, `invalid interface name "../port1"`)
}

func TestSplitName(t *testing.T) {
	iface, kind := splitName("42-metal-core-vni104001.netdev")
	require.Equal(t, "vni104001", iface)
	require.Equal(t, "netdev", kind)
	iface, kind = splitName("10-mgmt.netdev")
	require.Empty(t, iface)
	require.Empty(t, kind)
}

// TestRenderGolden compares the rendered files with the expected ones in the
// templates' test data, one directory per case.
func TestRenderGolden(t *testing.T) {
	const testData = "../templates/test_data"
	cases := map[string][2]string{
		"downports": {"../templates/test_down_interfaces/conf_with_downports.yaml", "../templates/test_down_interfaces/bisdn_networkd_with_downports"},
	}
	entries, err := os.ReadDir(testData)
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() {
			cases[e.Name()] = [2]string{filepath.Join(testData, e.Name(), "conf.yaml"), filepath.Join(testData, e.Name(), "bisdn_networkd")}
		}
	}

	tpl := templates.BISDNNetworkdTemplates("")
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(c[0])
			require.NoError(t, err)
			cfg := &types.Conf{}
			require.NoError(t, yaml.Unmarshal(data, cfg)) // nolint:musttag
			cfg.CapitalizeVrfName()

			files, err := render(tpl, cfg)
			require.NoError(t, err)

			expected, err := readManagedFiles(c[1])
			require.NoError(t, err)
			entries, err := os.ReadDir(c[1])
			require.NoError(t, err)
			require.Len(t, expected, len(entries), "all expected files must be managed files")

			got, want := map[string]string{}, map[string]string{}
			for k, v := range files {
				got[k] = string(v)
			}
			for k, v := range expected {
				want[k] = string(v)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s render differs:%s", c[1], diff)
			}
		})
	}
}

func TestApplyRejectsBladePorts(t *testing.T) {
	a, _, reloads := newTestApplier(t)
	m := &BISDN{networkdApplier: a, log: slog.Default()}

	cfg := testConf()
	cfg.Ports.BladePorts = []string{"port5"}
	require.ErrorContains(t, m.Apply(context.Background(), cfg), "blade ports are not supported")
	require.Zero(t, *reloads)
}
