package bisdn

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const bisdnOSRelease = `ID=bisdn-linux
NAME="BISDN Linux"
VERSION="6.0.0 (Diavola)"
VERSION_ID=6.0.0
PRETTY_NAME="BISDN Linux 6.0.0 (Diavola)"
`

func TestIsBISDN(t *testing.T) {
	dir := t.TempDir()
	bisdnFile := filepath.Join(dir, "bisdn")
	require.NoError(t, os.WriteFile(bisdnFile, []byte(bisdnOSRelease), 0o644))
	debianFile := filepath.Join(dir, "debian")
	require.NoError(t, os.WriteFile(debianFile, []byte("ID=debian\nVERSION_ID=\"13\"\n"), 0o644))

	require.True(t, IsBISDN(bisdnFile))
	require.False(t, IsBISDN(debianFile))
	require.False(t, IsBISDN(filepath.Join(dir, "missing")))
}

func TestOSRelease(t *testing.T) {
	v := osRelease(strings.NewReader("# comment\n" + bisdnOSRelease))
	require.Equal(t, "bisdn-linux", v["ID"])
	require.Equal(t, "6.0.0", v["VERSION_ID"])
	require.Equal(t, "BISDN Linux 6.0.0 (Diavola)", v["PRETTY_NAME"])
}

func TestFrontPanel(t *testing.T) {
	for name, want := range map[string]bool{
		"port1": true, "port32": true, "port130": true,
		"enp0": false, "mgmt": false, "port": false, "port1.10": false, "bridge": false, "Ethernet0": false,
	} {
		require.Equal(t, want, frontPanel.MatchString(name), name)
	}
	// numeric, not lexical order
	require.Less(t, portNumber("port2"), portNumber("port10"))
}

func TestSanitizeConfig(t *testing.T) {
	b := New(slog.Default(), "", "", "")
	cfg := &types.Conf{Ports: types.Ports{Vrfs: map[string]*types.Vrf{"vrf104001": {VNI: 104001}}}}
	b.SanitizeConfig(cfg)
	require.Equal(t, DefaultManagementInterface, cfg.Ports.Eth0.Name)
	require.Contains(t, cfg.Ports.Vrfs, "Vrf104001")

	b = New(slog.Default(), "", "", "eth1")
	b.SanitizeConfig(cfg)
	require.Equal(t, "eth1", cfg.Ports.Eth0.Name)
}
