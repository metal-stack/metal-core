//go:build client && boundary

package core

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/metal-stack/api/go/client"
	adminv2 "github.com/metal-stack/api/go/metalstack/admin/v2"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

// TestAPIBoundary is launched by metal-apiserver's TestMetalCoreBoundary. All API
// calls use its real HTTP server, validation, authentication and repositories.
// Only the local switch backend and host network observations are substituted.
func TestAPIBoundary(t *testing.T) {
	baseURL := os.Getenv("METAL_CORE_BOUNDARY_URL")
	adminToken := os.Getenv("METAL_CORE_BOUNDARY_ADMIN_TOKEN")
	require.NotEmpty(t, baseURL, "run make test-metal-core-boundary in the metal-apiserver checkout")
	require.NotEmpty(t, adminToken)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	newClient := func(token string) client.Client {
		t.Helper()
		c, err := client.New(&client.DialConfig{BaseURL: baseURL, Token: token, Log: log, UserAgent: "metal-core-boundary-test"})
		require.NoError(t, err)
		return c
	}
	admin := newClient(adminToken)
	ctx := t.Context()

	// Configure through admin RPCs, including URL validation against a local artifact server.
	artifacts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(artifacts.Close)
	const partitionID = "boundary-partition"
	rdnss := []string{"fd00:20:ffff::53"}
	p, err := admin.Adminv2().Partition().Create(ctx, &adminv2.PartitionServiceCreateRequest{
		Partition: &apiv2.Partition{Id: partitionID, BootConfiguration: &apiv2.PartitionBootConfiguration{
			ImageUrl: artifacts.URL, KernelUrl: artifacts.URL, Rdnss: rdnss,
		}},
	})
	require.NoError(t, err)
	// The core must work with an infra-only token, not the administrator's token.
	tok, err := admin.Adminv2().Token().Create(ctx, &adminv2.TokenServiceCreateRequest{
		User: new("metal-core-boundary"),
		TokenCreateRequest: &apiv2.TokenServiceCreateRequest{
			Description: "metal-core boundary test", Expires: durationpb.New(10 * time.Minute),
			InfraRole: apiv2.InfraRole_INFRA_ROLE_EDITOR.Enum(),
		},
	})
	require.NoError(t, err)
	infra := newClient(tok.Secret)
	_, err = infra.Adminv2().Partition().Capacity(ctx, &adminv2.PartitionServiceCapacityRequest{})
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	nos := &boundaryNOS{}
	c := New(Config{
		Log: log, ASN: "420000001", LoopbackIP: "10.0.0.1", CIDR: "192.0.2.2/24",
		PartitionID: partitionID, RackID: "boundary-rack", ManagementGateway: "192.0.2.1",
		PXEVlanID: 4000, ReconfigureSwitch: true, Client: infra, NOS: nos, Metrics: metrics.New(),
	})
	c.fillManagementInfo = func(cfg *types.Conf, gateway string) error {
		cfg.Ports.Eth0 = types.Nic{AddressCIDR: "192.0.2.2/24", Gateway: gateway}
		return nil
	}
	c.linkStatus = func(string) (apiv2.SwitchPortStatus, error) {
		return apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP, nil
	}
	c.vlanMapping = func() (vlan.Mapping, error) { return vlan.Mapping{}, nil }
	require.NoError(t, c.RegisterSwitch(ctx, 5*time.Second))
	hostname, err := os.Hostname()
	require.NoError(t, err)
	getSwitch := func() (*apiv2.Switch, error) {
		resp, err := infra.Infrav2().Switch().Get(ctx, &infrav2.SwitchServiceGetRequest{Id: hostname})
		if err != nil {
			return nil, err
		}
		return resp.Switch, nil
	}

	// Initially the API has no boot network, so core configures PXE. Creating the
	// network below must migrate this same core instance without a local mode flag.
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := getSwitch()
		if !assert.NoError(ct, err) {
			return
		}
		assert.Nil(ct, sw.BootVni)
		cfg := nos.applied()
		if !assert.NotNil(ct, cfg) {
			return
		}
		assert.NotContains(ct, cfg.Ports.Vrfs, types.BootVrfName)
		assert.Equal(ct, uint16(4000), cfg.PXEVlanID)
		assert.Len(ct, cfg.Ports.Unprovisioned, 2)
		for _, port := range cfg.Ports.Unprovisioned {
			assert.False(ct, port.BootPrefix.IsValid())
		}
	})
	_, err = admin.Adminv2().Network().Create(ctx, &adminv2.NetworkServiceCreateRequest{
		Id:                       new("boundary-boot"),
		Partition:                new(partitionID),
		Type:                     apiv2.NetworkType_NETWORK_TYPE_BOOT,
		Prefixes:                 []string{"fd00:20::/62"},
		Vrf:                      new(uint32(42)),
		DefaultChildPrefixLength: &apiv2.ChildPrefixLength{Ipv6: new(uint32(64))},
	})
	require.NoError(t, err)

	pool := netip.MustParsePrefix("fd00:20::/48")
	assertPrefixes := func(ct *assert.CollectT, sw *apiv2.Switch) map[string]netip.Prefix {
		got := map[string]netip.Prefix{}
		seen := map[netip.Prefix]string{}
		for _, nic := range sw.Nics {
			assert.Equal(ct, types.BootVrfName, nic.GetVrf())
			prefix, err := netip.ParsePrefix(nic.GetBootPrefix())
			if !assert.NoError(ct, err) {
				continue
			}
			assert.Equal(ct, 64, prefix.Bits())
			assert.Equal(ct, prefix.Masked(), prefix)
			assert.True(ct, pool.Contains(prefix.Addr()), "boot prefix must belong to the partition pool")
			assert.NotContains(ct, seen, prefix, "active ports must have distinct boot prefixes")
			seen[prefix] = nic.Name
			got[nic.Name] = prefix
		}
		assert.Len(ct, got, 2)
		assert.Contains(ct, got, "Ethernet0")
		assert.Contains(ct, got, "Ethernet4")
		return got
	}
	checkApplied := func(wantRDNSS []string, since time.Time) func(*assert.CollectT) {
		return func(ct *assert.CollectT) {
			sw, err := getSwitch()
			if !assert.NoError(ct, err) {
				return
			}
			prefixes := assertPrefixes(ct, sw)
			assert.Equal(ct, uint32(42), sw.GetBootVni())
			assert.Equal(ct, wantRDNSS, sw.BootRdnss)
			if assert.NotNil(ct, sw.LastSync) {
				assert.True(ct, sw.LastSync.Time.AsTime().After(since), "a successful heartbeat must follow this change")
				assert.Empty(ct, sw.LastSync.GetError())
			}
			for _, nic := range sw.Nics {
				assert.Equal(ct, apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP, nic.GetState().GetActual())
			}
			cfg := nos.applied()
			if !assert.NotNil(ct, cfg) {
				return
			}
			assert.Equal(ct, hostname, cfg.Name)
			assert.Equal(ct, wantRDNSS, cfg.BootRDNSS)
			if assert.Contains(ct, cfg.Ports.Vrfs, types.BootVrfName) {
				assert.Equal(ct, uint32(42), cfg.Ports.Vrfs[types.BootVrfName].VNI)
				assert.NotZero(ct, cfg.Ports.Vrfs[types.BootVrfName].VLANID)
			}
			got := map[string]netip.Prefix{}
			for name, port := range cfg.Ports.Unprovisioned {
				got[name] = port.BootPrefix
			}
			assert.Equal(ct, prefixes, got)
		}
	}

	// Exercise the production polling/apply/heartbeat loop, not a test copy of it.
	runBoundaryLoop(t, c, checkApplied(rdnss, time.Now()))

	// Re-registration must keep valid allocations, and partition changes must reach Apply.
	require.NoError(t, c.RegisterSwitch(ctx, 5*time.Second))
	rdnss = []string{"fd00:20:ffff::54"}
	_, err = admin.Adminv2().Partition().Update(ctx, &adminv2.PartitionServiceUpdateRequest{
		Id: partitionID, UpdateMeta: &apiv2.UpdateMeta{UpdatedAt: p.Partition.Meta.UpdatedAt},
		BootConfiguration: &apiv2.PartitionBootConfiguration{
			ImageUrl: artifacts.URL, KernelUrl: artifacts.URL, Rdnss: rdnss,
		},
	})
	require.NoError(t, err)
	runBoundaryLoop(t, c, checkApplied(rdnss, time.Now()))

	// A failed backend apply must produce an error heartbeat, then recover on a later tick.
	nos.failWith(errors.New("boundary apply failure"))
	since := time.Now()
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := getSwitch()
		if !assert.NoError(ct, err) || !assert.NotNil(ct, sw.LastSyncError) {
			return
		}
		assert.Contains(ct, sw.LastSyncError.GetError(), "boundary apply failure")
		assert.True(ct, sw.LastSyncError.Time.AsTime().After(since))
		assertPrefixes(ct, sw)
	})
	nos.failWith(nil)
	runBoundaryLoop(t, c, checkApplied(rdnss, time.Now()))

	testBootPrefixReuseBoundary(t, admin, infra, artifacts.URL)
	testProvisioningBoundary(t, c, nos, admin, infra, artifacts.URL)
	// Both ports must return to boot configuration using their current API prefixes.
	runBoundaryLoop(t, c, checkApplied(rdnss, time.Now()))

	// Drained replacement must retain old pools until a successful enabled apply.
	partitionSwitches, err := admin.Adminv2().Switch().List(ctx, &adminv2.SwitchServiceListRequest{Query: &apiv2.SwitchQuery{Partition: new(partitionID)}})
	require.NoError(t, err)
	for _, sw := range partitionSwitches.Switches {
		states := map[string]apiv2.SwitchPortStatus{}
		for _, nic := range sw.Nics {
			nic.State.Desired = apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN.Enum()
			states[nic.Name] = apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN
		}
		_, err = admin.Adminv2().Switch().Update(ctx, &adminv2.SwitchServiceUpdateRequest{
			Id: sw.Id, UpdateMeta: &apiv2.UpdateMeta{UpdatedAt: sw.Meta.UpdatedAt}, Nics: sw.Nics,
		})
		require.NoError(t, err)
		if sw.Id != hostname {
			_, err = infra.Infrav2().Switch().Heartbeat(ctx, &infrav2.SwitchServiceHeartbeatRequest{Id: sw.Id, PortStates: states})
			require.NoError(t, err)
		}
	}
	c.linkStatus = func(string) (apiv2.SwitchPortStatus, error) {
		return apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN, nil
	}
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := getSwitch()
		if !assert.NoError(ct, err) {
			return
		}
		for _, nic := range sw.Nics {
			assert.Equal(ct, apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN, nic.State.Actual)
			assert.Equal(ct, apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN, nic.State.GetDesired())
		}
	})
	_, err = admin.Adminv2().Network().Delete(ctx, &adminv2.NetworkServiceDeleteRequest{Id: "boundary-boot"})
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	replacement, err := admin.Adminv2().Network().Update(ctx, &adminv2.NetworkServiceUpdateRequest{
		Id: "boundary-boot", UpdateMeta: &apiv2.UpdateMeta{LockingStrategy: apiv2.OptimisticLockingStrategy_OPTIMISTIC_LOCKING_STRATEGY_SERVER}, Prefixes: []string{"fd00:30::/60"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, replacement.Network.RetiringBootPrefixes)
	checkRetained := func() {
		t.Helper()
		nw, err := admin.Adminv2().Network().Get(ctx, &adminv2.NetworkServiceGetRequest{Id: "boundary-boot"})
		require.NoError(t, err)
		require.Equal(t, replacement.Network.RetiringBootPrefixes, nw.Network.RetiringBootPrefixes)
	}
	c.enableReconfigureSwitch = false
	since = time.Now()
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := getSwitch()
		if !assert.NoError(ct, err) || !assert.NotNil(ct, sw.LastSync) {
			return
		}
		assert.True(ct, sw.LastSync.Time.AsTime().After(since))
	})
	checkRetained()
	c.enableReconfigureSwitch = true
	nos.failWith(errors.New("replacement apply failure"))
	since = time.Now()
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := getSwitch()
		if !assert.NoError(ct, err) || !assert.NotNil(ct, sw.LastSyncError) {
			return
		}
		assert.Contains(ct, sw.LastSyncError.GetError(), "replacement apply failure")
		assert.True(ct, sw.LastSyncError.Time.AsTime().After(since))
	})
	checkRetained()
	for _, sw := range partitionSwitches.Switches {
		if sw.Id == hostname {
			continue
		}
		peerCore := *c
		peerCore.nos = &boundaryNOS{}
		applied, err := peerCore.reconfigureSwitch(ctx, sw.Id)
		require.NoError(t, err)
		_, err = infra.Infrav2().Switch().Heartbeat(ctx, &infrav2.SwitchServiceHeartbeatRequest{Id: sw.Id, AppliedBootNetworkRevision: applied.BootNetworkRevision})
		require.NoError(t, err)
	}
	checkRetained()
	nos.failWith(nil)
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		nw, err := admin.Adminv2().Network().Get(ctx, &adminv2.NetworkServiceGetRequest{Id: "boundary-boot"})
		if !assert.NoError(ct, err) {
			return
		}
		assert.Empty(ct, nw.Network.RetiringBootPrefixes)
		cfg := nos.applied()
		if !assert.NotNil(ct, cfg) {
			return
		}
		for _, port := range cfg.Ports.Unprovisioned {
			assert.True(ct, netip.MustParsePrefix("fd00:30::/60").Contains(port.BootPrefix.Addr()))
		}
	})
}

func runBoundaryLoop(t *testing.T, c *Core, check func(*assert.CollectT)) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.ConstantlyReconfigureSwitch(ctx, 100*time.Millisecond, 3*time.Second)
	}()
	defer func() {
		cancel()
		<-done
	}()
	require.EventuallyWithT(t, check, 15*time.Second, 100*time.Millisecond)
}

type boundaryNOS struct {
	switcher.NOS
	mu       sync.Mutex
	last     *types.Conf
	applyErr error
}

func (n *boundaryNOS) SanitizeConfig(cfg *types.Conf) { cfg.CapitalizeVrfName() }
func (n *boundaryNOS) Apply(_ context.Context, cfg *types.Conf) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.applyErr != nil {
		return n.applyErr
	}
	n.last = cfg
	return nil
}
func (n *boundaryNOS) applied() *types.Conf {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.last
}
func (n *boundaryNOS) failWith(err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.applyErr = err
}
func (*boundaryNOS) IsInitialized(context.Context) (bool, error) { return true, nil }
func (*boundaryNOS) GetNics(context.Context, []string) ([]*apiv2.SwitchNic, error) {
	return []*apiv2.SwitchNic{
		{Name: "Ethernet0", Identifier: "Ethernet0", State: &apiv2.NicState{Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN}},
		{Name: "Ethernet4", Identifier: "Ethernet4", State: &apiv2.NicState{Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN}},
	}, nil
}
func (*boundaryNOS) GetOS() (*apiv2.SwitchOS, error) {
	return &apiv2.SwitchOS{Vendor: apiv2.SwitchOSVendor_SWITCH_OS_VENDOR_SONIC, Version: "202411"}, nil
}
func (*boundaryNOS) GetManagement() (string, string, error) { return "192.0.2.2", "admin", nil }
