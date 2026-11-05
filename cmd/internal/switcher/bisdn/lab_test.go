//go:build lab

package bisdn

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

// TestLabApply applies the switch config in BISDN_LAB_CONF to the running
// system, as metal-core's reconfigure loop does. bsidn-test's conformance lab builds it
// with go test -c -tags lab and runs it in its BISDN Linux leaves.
func TestLabApply(t *testing.T) {
	path := os.Getenv("BISDN_LAB_CONF")
	if path == "" {
		t.Skip("BISDN_LAB_CONF not set")
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cfg := &types.Conf{}
	require.NoError(t, yaml.Unmarshal(data, cfg)) // nolint:musttag

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	nos := New(log, "", "", os.Getenv("BISDN_LAB_MGMT_INTERFACE"))

	nos.SanitizeConfig(cfg)
	require.NoError(t, cfg.FillRouteMapsAndIPPrefixLists())
	// VRFs with a preset VLAN ID keep it, so lab checks can rely on fixed
	// VLAN numbers; the others get one as in metal-core's reconfigure loop.
	m, err := vlan.ReadMapping()
	require.NoError(t, err)
	unset := types.Vrfs{}
	for name, vrf := range cfg.Ports.Vrfs {
		if vrf.VLANID == 0 {
			unset[name] = vrf
		} else {
			m[vrf.VLANID] = vrf.VNI
		}
	}
	rest := &types.Conf{Ports: types.Ports{Vrfs: unset}}
	require.NoError(t, rest.FillVLANIDs(m))

	require.NoError(t, nos.Apply(context.Background(), cfg))
}
