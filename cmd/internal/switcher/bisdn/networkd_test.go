package bisdn

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

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
		tpl:   templates.BISDNNetworkdTemplate(""),
		links: links,
		reload: func(context.Context) error {
			reloads++
			return nil
		},
		log: slog.Default(),
	}, links, &reloads
}

func TestNetworkdApply(t *testing.T) {
	ctx := context.Background()
	a, links, reloads := newTestApplier(t)

	foreign := filepath.Join(a.dir, "10-mgmt.netdev")
	require.NoError(t, os.WriteFile(foreign, []byte("[NetDev]\nName=mgmt\nKind=vrf\n"), 0644))

	cfg := testConf()
	require.NoError(t, a.Apply(ctx, cfg))
	require.Equal(t, 1, *reloads)
	require.Empty(t, links.deleted)
	require.Empty(t, links.released)

	files, err := readManagedFiles(a.dir)
	require.NoError(t, err)
	require.Contains(t, files, "30-bisdn-Vrf104001.netdev")
	require.Contains(t, files, "42-bisdn-vni104000.netdev")
	require.Contains(t, files, "50-bisdn-port2.network")
	require.Contains(t, string(files["50-bisdn-port2.network"]), "VRF=Vrf104001\n")

	// unchanged config: no reload
	require.NoError(t, a.Apply(ctx, testConf()))
	require.Equal(t, 1, *reloads)

	// the machine on port2 is freed and the tenant VRF goes away
	cfg = testConf()
	cfg.Ports.Unprovisioned = append(cfg.Ports.Unprovisioned, "port2")
	cfg.Ports.Vrfs = types.Vrfs{}
	require.NoError(t, a.Apply(ctx, cfg))
	require.Equal(t, 2, *reloads)
	slices.Sort(links.deleted)
	require.Equal(t, []string{"Vrf104001", "vlan1001", "vni104001"}, links.deleted)
	require.Empty(t, links.released, "port2 still has a .network file")

	files, err = readManagedFiles(a.dir)
	require.NoError(t, err)
	require.NotContains(t, files, "30-bisdn-Vrf104001.netdev")
	require.Contains(t, string(files["51-bisdn-port2.network"]), "Bridge=bridge\n")

	// port4 is no longer configured at all
	links.deleted = nil
	cfg.Ports.Unprovisioned = []string{"port2"}
	require.NoError(t, a.Apply(ctx, cfg))
	require.Empty(t, links.deleted)
	require.Equal(t, []string{"port4"}, links.released)

	// a new loopback changes the VXLAN source, so the VTEP is created again
	links.released = nil
	cfg.Loopback = "10.0.0.11"
	require.NoError(t, a.Apply(ctx, cfg))
	require.Equal(t, []string{"vni104000"}, links.deleted)
	require.Empty(t, links.released)

	_, err = os.Stat(foreign)
	require.NoError(t, err, "files that metal-core doesn't own must stay")
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

func TestSplitFiles(t *testing.T) {
	files, err := splitFiles([]byte("#> 10-bisdn-lo.network\n[Match]\nName=lo\n\n#> 40-bisdn-bridge.netdev\n[NetDev]\nName=bridge\n"))
	require.NoError(t, err)
	require.Equal(t, map[string][]byte{
		"10-bisdn-lo.network":    []byte("[Match]\nName=lo\n"),
		"40-bisdn-bridge.netdev": []byte("[NetDev]\nName=bridge\n"),
	}, files)

	for _, bad := range []string{
		"[Match]\nName=lo\n",
		"#> lo.network\n[Match]\n",
		"#> 10-bisdn-../x.network\n",
		"#> 10-bisdn-lo.network\n#> 10-bisdn-lo.network\n",
	} {
		_, err := splitFiles([]byte(bad))
		require.Error(t, err, bad)
	}
}

func TestIniValue(t *testing.T) {
	content := []byte("[Match]\nName=port2\n\n[NetDev]\nKind=vrf\nName = Vrf1\n")
	require.Equal(t, "port2", iniValue(content, "Match", "Name"))
	require.Equal(t, "Vrf1", iniValue(content, "NetDev", "Name"))
	require.Empty(t, iniValue(content, "Network", "Name"))
}

func TestApplyRejectsBladePorts(t *testing.T) {
	a, _, reloads := newTestApplier(t)
	m := &BISDN{networkdApplier: a, log: slog.Default()}

	cfg := testConf()
	cfg.Ports.BladePorts = []string{"port5"}
	require.ErrorContains(t, m.Apply(context.Background(), cfg), "blade ports are not supported")
	require.Zero(t, *reloads)
}
