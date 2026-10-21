package redis

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/require"
)

func TestApplier_FirstApplyWithMatchingPortStatus(t *testing.T) {
	mr := miniredis.RunT(t)
	// Production connects over Unix sockets to standalone Redis.
	mr.Server().SetPreHook(func(peer *server.Peer, cmd string, _ ...string) bool {
		if cmd == "CLUSTER" {
			peer.WriteError("ERR This instance has cluster support disabled")
			return true
		}
		return false
	})
	sock := startUnixServer(t, mr.Server())
	var dbCfg db.Config
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, `{
		"DATABASES": {
			"APPL_DB": {"id": 0, "instance": "redis", "separator": ":"},
			"ASIC_DB": {"id": 1, "instance": "redis", "separator": ":"},
			"CONFIG_DB": {"id": 4, "instance": "redis", "separator": "|"},
			"COUNTERS_DB": {"id": 2, "instance": "redis", "separator": ":"}
		}, "INSTANCES": {"redis": {"unix_socket_path": %q}}
	}`, sock), &dbCfg))
	d, err := db.New(&dbCfg)
	require.NoError(t, err)
	a := NewApplier(slog.Default(), d)
	mr.Select(4)
	mr.HSet("PORT|Ethernet0", "admin_status", "up", "mtu", "1500")
	mr.HSet("VLAN|Vlan4000", "vlanid", "4000")
	mr.HSet("VLAN_MEMBER|Vlan4000|Ethernet0", "tagging_mode", "untagged")
	cfg := &types.Conf{PXEVlanID: 4000, Ports: types.Ports{
		Unprovisioned: []string{"Ethernet0"},
		AdminStatus:   map[string]types.PortStatus{"Ethernet0": types.PortStatusUp},
	}}
	require.NoError(t, a.Apply(t.Context(), cfg))
	require.Equal(t, "9000", mr.HGet("PORT|Ethernet0", "mtu"))
	require.Equal(t, "up", mr.HGet("PORT|Ethernet0", "admin_status"))

	// With unchanged config and matching admin status, the second apply skips reconciliation.
	mr.HSet("PORT|Ethernet0", "mtu", "1500")
	require.NoError(t, a.Apply(t.Context(), cfg))
	require.Equal(t, "1500", mr.HGet("PORT|Ethernet0", "mtu"))
}

func startUnixServer(t *testing.T, srv *server.Server) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "redis.sock")
	l, err := net.Listen("unix", sock)
	require.NoError(t, err)
	done := make(chan struct{})
	t.Cleanup(func() {
		_ = l.Close()
		// Stop accepting connections before miniredis shuts down.
		<-done
	})

	go func() {
		defer close(done)
		for {
			client, err := l.Accept()
			if err != nil {
				return
			}
			srv.ServeConn(client)
		}
	}()
	return sock
}
