//go:build client && boundary

package core

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/metal-stack/api/go/client"
	adminv2 "github.com/metal-stack/api/go/metalstack/admin/v2"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testProvisioningBoundary(t *testing.T, c *Core, nos *boundaryNOS, admin, infra client.Client, artifactURL string) {
	t.Helper()
	ctx := t.Context()
	const machineID = "00000000-0000-0000-0000-000000000020"
	before := nos.applied()
	hostname := before.Name
	peerID := hostname + "-peer"
	bootPrefix := before.Ports.Unprovisioned["Ethernet0"].BootPrefix
	idlePrefix := before.Ports.Unprovisioned["Ethernet4"].BootPrefix

	t.Log("provisioning: create allocation prerequisites through the API")
	tenant, err := admin.Apiv2().Tenant().Create(ctx, &apiv2.TenantServiceCreateRequest{Name: "boundary-tenant"})
	require.NoError(t, err)
	project, err := admin.Apiv2().Project().Create(ctx, &apiv2.ProjectServiceCreateRequest{Name: "boundary-project", Login: tenant.Tenant.Login})
	require.NoError(t, err)
	_, err = admin.Adminv2().Size().Create(ctx, &adminv2.SizeServiceCreateRequest{Size: &apiv2.Size{
		Id: "boundary-size", Constraints: []*apiv2.SizeConstraint{
			{Type: apiv2.SizeConstraintType_SIZE_CONSTRAINT_TYPE_CORES, Min: 4, Max: 4},
			{Type: apiv2.SizeConstraintType_SIZE_CONSTRAINT_TYPE_MEMORY, Min: 1024, Max: 1024},
			{Type: apiv2.SizeConstraintType_SIZE_CONSTRAINT_TYPE_STORAGE, Min: 1024, Max: 1 << 40},
		},
	}})
	require.NoError(t, err)
	_, err = admin.Adminv2().Filesystem().Create(ctx, &adminv2.FilesystemServiceCreateRequest{FilesystemLayout: &apiv2.FilesystemLayout{
		Id:          "boundary-fsl",
		Constraints: &apiv2.FilesystemLayoutConstraints{Sizes: []string{"boundary-size"}, Images: map[string]string{"debian": ">= 12.0"}},
		Disks: []*apiv2.Disk{{Device: "/dev/sda", Partitions: []*apiv2.DiskPartition{
			{Number: 0, Size: 1024, GptType: apiv2.GPTType_GPT_TYPE_LINUX.Enum()},
		}}},
	}})
	require.NoError(t, err)
	_, err = admin.Adminv2().Image().Create(ctx, &adminv2.ImageServiceCreateRequest{Image: &apiv2.Image{
		Id: "debian-12.0.1", Url: artifactURL,
		Features:       []apiv2.ImageFeature{apiv2.ImageFeature_IMAGE_FEATURE_MACHINE},
		Classification: apiv2.ImageClassification_IMAGE_CLASSIFICATION_SUPPORTED,
	}})
	require.NoError(t, err)
	_, err = admin.Adminv2().Network().Create(ctx, &adminv2.NetworkServiceCreateRequest{
		Id: new("boundary-super"), Partition: &c.partitionID, Prefixes: []string{"10.20.0.0/16"},
		Type:                     apiv2.NetworkType_NETWORK_TYPE_SUPER,
		DefaultChildPrefixLength: &apiv2.ChildPrefixLength{Ipv4: new(uint32(24))},
	})
	require.NoError(t, err)
	network, err := admin.Apiv2().Network().Create(ctx, &apiv2.NetworkServiceCreateRequest{
		Project: project.Project.Uuid, Name: new("boundary-tenant-network"), Partition: &c.partitionID,
	})
	require.NoError(t, err)
	require.NotNil(t, network.Network.Vrf)
	tenantVNI := network.Network.GetVrf()
	tenantVRF := fmt.Sprintf("Vrf%d", tenantVNI)

	// Hardware registration requires a pair of switches in the same rack. Only
	// the primary runs Core; the peer is registered as inventory over the same API.
	peerNics, err := nos.GetNics(ctx, nil)
	require.NoError(t, err)
	peer, err := infra.Infrav2().Switch().Register(ctx, &infrav2.SwitchServiceRegisterRequest{Switch: &apiv2.Switch{
		Id: peerID, Partition: c.partitionID, Rack: &c.rackID, ManagementIp: "192.0.2.3", Nics: peerNics,
		Os: &apiv2.SwitchOS{Vendor: apiv2.SwitchOSVendor_SWITCH_OS_VENDOR_SONIC, Version: "202411", MetalCoreVersion: "boundary-test"},
	}})
	require.NoError(t, err)
	released := []string{bootPrefix.String()}
	for _, nic := range peer.Switch.Nics {
		if nic.Name == "Ethernet0" {
			released = append(released, nic.GetBootPrefix())
		}
	}
	require.Len(t, released, 2)
	_, err = infra.Infrav2().Boot().Booting(ctx, &infrav2.BootServiceBootingRequest{Uuid: machineID, Partition: c.partitionID})
	require.NoError(t, err)
	sendEvent := func(event apiv2.MachineProvisioningEventType) {
		t.Helper()
		resp, err := infra.Infrav2().Event().Send(ctx, &infrav2.EventServiceSendRequest{Events: map[string]*apiv2.MachineProvisioningEvent{
			machineID: {Event: event, Time: timestamppb.Now(), Message: "boundary provisioning"},
		}})
		require.NoError(t, err)
		require.Equal(t, uint64(1), resp.Events)
		require.Empty(t, resp.Failed)
	}
	sendEvent(apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_PREPARING)
	sendEvent(apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_REGISTERING)

	// Hammer calls use a machine-scoped token, distinct from Core's infra token.
	tok, err := admin.Adminv2().Token().Create(ctx, &adminv2.TokenServiceCreateRequest{
		User: new("metal-core-boundary"), TokenCreateRequest: &apiv2.TokenServiceCreateRequest{
			Description: "boundary hammer", Expires: durationpb.New(10 * time.Minute),
			MachineRoles: map[string]apiv2.MachineRole{machineID: apiv2.MachineRole_MACHINE_ROLE_EDITOR},
		},
	})
	require.NoError(t, err)
	hammer, err := client.New(&client.DialConfig{BaseURL: os.Getenv("METAL_CORE_BOUNDARY_URL"), Token: tok.Secret, Log: c.log})
	require.NoError(t, err)
	registered, err := hammer.Infrav2().Boot().Register(ctx, &infrav2.BootServiceRegisterRequest{
		Uuid: machineID, Partition: c.partitionID, Bios: &apiv2.MachineBios{Version: "1.0"},
		Bmc: &apiv2.MachineBMC{Address: "192.0.2.10:623", Mac: "02:00:00:00:02:01", User: "boundary", Password: "test-password"},
		Hardware: &apiv2.MachineHardware{
			Memory: 1024, Cpus: []*apiv2.MetalCPU{{Cores: 4}},
			Disks: []*apiv2.MachineBlockDevice{{Name: "/dev/sda", Size: 10 << 30}},
			Nics: []*apiv2.MachineNic{
				{Name: "lan0", Mac: "02:00:00:00:00:01", Neighbors: []*apiv2.MachineNic{{Name: "Ethernet0", Identifier: "Ethernet0", Hostname: hostname, Mac: "02:00:00:00:01:01"}}},
				{Name: "lan1", Mac: "02:00:00:00:00:02", Neighbors: []*apiv2.MachineNic{{Name: "Ethernet0", Identifier: "Ethernet0", Hostname: peerID, Mac: "02:00:00:00:01:02"}}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "boundary-size", registered.Size)
	sendEvent(apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_WAITING)

	allocationResult := waitBoundaryAllocation(t, hammer, machineID)
	// Observe the server's waiting flag instead of sleeping before allocation.
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		resp, err := admin.Adminv2().Machine().List(ctx, &adminv2.MachineServiceListRequest{
			Query: &apiv2.MachineQuery{Uuid: new(machineID), Waiting: new(true)},
		})
		if assert.NoError(ct, err) {
			assert.Len(ct, resp.Machines, 1)
		}
	}, 10*time.Second, 100*time.Millisecond)
	allocated, err := admin.Apiv2().Machine().Create(ctx, &apiv2.MachineServiceCreateRequest{
		Uuid: new(machineID), Project: project.Project.Uuid, Name: "boundary-machine", Hostname: new("boundary-machine"),
		Image: "debian-12.0.1", AllocationType: apiv2.MachineAllocationType_MACHINE_ALLOCATION_TYPE_MACHINE,
		Networks: []*apiv2.MachineAllocationNetwork{{Network: network.Network.Id}},
	})
	require.NoError(t, err)
	select {
	case result := <-allocationResult:
		require.NoError(t, result.err)
		require.NotNil(t, result.allocation)
		require.Equal(t, allocated.Machine.Allocation.Uuid, result.allocation.Uuid)
	case <-time.After(10 * time.Second):
		t.Fatal("hammer did not receive the allocation")
	}
	sendEvent(apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_INSTALLING)

	// Observe and acknowledge both lifecycle commands through the real BMC stream.
	bmcCommands := acknowledgeBoundaryBMC(t, infra, c.partitionID, machineID)
	t.Log("provisioning: InstallationSucceeded moves the port into the tenant VRF")
	_, err = hammer.Infrav2().Boot().InstallationSucceeded(ctx, &infrav2.BootServiceInstallationSucceededRequest{
		Uuid: machineID, ConsolePassword: "boundary-console",
	})
	require.NoError(t, err)
	requireBoundaryCommand(t, bmcCommands, apiv2.MachineBMCCommand_MACHINE_BMC_COMMAND_MACHINE_CREATED)
	since := time.Now()
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := infra.Infrav2().Switch().Get(ctx, &infrav2.SwitchServiceGetRequest{Id: hostname})
		if !assert.NoError(ct, err) {
			return
		}
		for _, nic := range sw.Switch.Nics {
			if nic.Name == "Ethernet0" {
				assert.Equal(ct, fmt.Sprintf("vrf%d", tenantVNI), nic.GetVrf())
				assert.Empty(ct, nic.GetBootPrefix(), "entering the tenant VRF releases the boot reservation")
			}
		}
		if assert.NotNil(ct, sw.Switch.LastSync) {
			assert.True(ct, sw.Switch.LastSync.Time.AsTime().After(since))
			assert.Empty(ct, sw.Switch.LastSync.GetError())
		}
		cfg := nos.applied()
		assert.NotContains(ct, cfg.Ports.Unprovisioned, "Ethernet0")
		if assert.Contains(ct, cfg.Ports.Vrfs, tenantVRF) {
			assert.Equal(ct, tenantVNI, cfg.Ports.Vrfs[tenantVRF].VNI)
			assert.Equal(ct, []string{"Ethernet0"}, cfg.Ports.Vrfs[tenantVRF].Neighbors)
		}
		assert.Contains(ct, cfg.Ports.Vrfs, types.BootVrfName)
		if assert.Contains(ct, cfg.Ports.Unprovisioned, "Ethernet4") {
			assert.Equal(ct, idlePrefix, cfg.Ports.Unprovisioned["Ethernet4"].BootPrefix)
		}
	})

	// The original /62 was full. A third switch can now consume exactly the two
	// prefixes released by provisioning, regardless of the allocator's ordering.
	t.Log("provisioning: released prefixes are reusable by ports on another switch")
	recipient, err := infra.Infrav2().Switch().Register(ctx, &infrav2.SwitchServiceRegisterRequest{Switch: &apiv2.Switch{
		Id: "provisioning-prefix-recipient", Partition: c.partitionID, Rack: new("recipient-rack"), ManagementIp: "192.0.2.4",
		Nics: []*apiv2.SwitchNic{
			{Name: "Ethernet8", Identifier: "Ethernet8", State: &apiv2.NicState{Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP}},
			{Name: "Ethernet12", Identifier: "Ethernet12", State: &apiv2.NicState{Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP}},
		},
		Os: peer.Switch.Os,
	}})
	require.NoError(t, err)
	var reused []string
	for _, nic := range recipient.Switch.Nics {
		reused = append(reused, nic.GetBootPrefix())
	}
	require.ElementsMatch(t, released, reused)
	// Re-registration and heartbeat must not reserve prefixes for tenant ports,
	// even with a full boot pool.
	require.NoError(t, c.RegisterSwitch(ctx, 3*time.Second))
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		sw, err := infra.Infrav2().Switch().Get(ctx, &infrav2.SwitchServiceGetRequest{Id: hostname})
		if !assert.NoError(ct, err) {
			return
		}
		for _, nic := range sw.Switch.Nics {
			if nic.Name == "Ethernet0" {
				assert.Empty(ct, nic.GetBootPrefix())
				assert.Equal(ct, fmt.Sprintf("vrf%d", tenantVNI), nic.GetVrf())
			}
		}
		if assert.NotNil(ct, sw.Switch.LastSync) {
			assert.Empty(ct, sw.Switch.LastSync.GetError())
		}
	})
	// Add capacity for reclaim while the old prefixes remain in use elsewhere.
	_, err = admin.Adminv2().Network().Update(ctx, &adminv2.NetworkServiceUpdateRequest{
		Id: "boundary-boot", UpdateMeta: &apiv2.UpdateMeta{LockingStrategy: apiv2.OptimisticLockingStrategy_OPTIMISTIC_LOCKING_STRATEGY_SERVER},
		Prefixes: []string{"fd00:20::/62", "fd00:20:0:4::/62"},
	})
	require.NoError(t, err)

	t.Log("provisioning: Core phone-home records the provisioned machine event")
	sendEvent(apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_BOOTING_NEW_KERNEL)
	phoneTime := time.Now().UTC()
	const phoneMessage = "provisioned boundary machine"
	c.phoneHome(ctx, []phoneHomeMessage{{machineID: machineID, payload: phoneMessage, time: phoneTime}})
	phoneCompleted := time.Now().UTC()
	m, err := admin.Adminv2().Machine().Get(ctx, &adminv2.MachineServiceGetRequest{Uuid: machineID})
	require.NoError(t, err)
	events := m.Machine.RecentProvisioningEvents
	require.NotNil(t, events)
	require.NotEmpty(t, events.Events)
	require.Equal(t, apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_PHONED_HOME, events.Events[0].Event)
	require.Equal(t, phoneMessage, events.Events[0].Message)
	// SendEvent stamps server receipt time rather than persisting the client time.
	// RethinkDB stores timestamps at millisecond precision.
	recorded := events.Events[0].Time.AsTime()
	require.False(t, recorded.Before(phoneTime.Truncate(time.Millisecond)))
	require.False(t, recorded.After(phoneCompleted))
	require.Nil(t, events.LastErrorEvent)
	require.Equal(t, apiv2.MachineLiveliness_MACHINE_LIVELINESS_ALIVE, m.Machine.Status.Liveliness)

	t.Log("provisioning: deallocation acknowledges BMC and acquires an available boot prefix")
	_, err = admin.Apiv2().Machine().Delete(ctx, &apiv2.MachineServiceDeleteRequest{Uuid: machineID, Project: project.Project.Uuid})
	require.NoError(t, err)
	requireBoundaryCommand(t, bmcCommands, apiv2.MachineBMCCommand_MACHINE_BMC_COMMAND_MACHINE_DELETED)
	since = time.Now()
	runBoundaryLoop(t, c, func(ct *assert.CollectT) {
		m, err := admin.Adminv2().Machine().Get(ctx, &adminv2.MachineServiceGetRequest{Uuid: machineID})
		if assert.NoError(ct, err) {
			assert.Nil(ct, m.Machine.Allocation)
			events := m.Machine.GetRecentProvisioningEvents().GetEvents()
			if assert.NotEmpty(ct, events) {
				assert.Equal(ct, apiv2.MachineProvisioningEventType_MACHINE_PROVISIONING_EVENT_TYPE_MACHINE_RECLAIM, events[0].Event)
			}
		}
		sw, err := infra.Infrav2().Switch().Get(ctx, &infrav2.SwitchServiceGetRequest{Id: hostname})
		var currentPrefix netip.Prefix
		if assert.NoError(ct, err) {
			for _, nic := range sw.Switch.Nics {
				if nic.Name == "Ethernet0" {
					assert.Empty(ct, nic.GetVrf())
					currentPrefix, err = netip.ParsePrefix(nic.GetBootPrefix())
					assert.NoError(ct, err)
					assert.Equal(ct, 64, currentPrefix.Bits())
					assert.True(ct, netip.MustParsePrefix("fd00:20:0:4::/62").Contains(currentPrefix.Addr()))
					assert.NotContains(ct, reused, nic.GetBootPrefix())
				}
			}
			if assert.NotNil(ct, sw.Switch.LastSync) {
				assert.True(ct, sw.Switch.LastSync.Time.AsTime().After(since))
				assert.Empty(ct, sw.Switch.LastSync.GetError())
			}
		}
		cfg := nos.applied()
		assert.NotContains(ct, cfg.Ports.Vrfs, tenantVRF)
		if assert.Contains(ct, cfg.Ports.Unprovisioned, "Ethernet0") {
			assert.Equal(ct, currentPrefix, cfg.Ports.Unprovisioned["Ethernet0"].BootPrefix)
		}
	})
}

type boundaryAllocationResult struct {
	allocation *apiv2.MachineAllocation
	err        error
}

func waitBoundaryAllocation(t *testing.T, hammer client.Client, machineID string) <-chan boundaryAllocationResult {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	result := make(chan boundaryAllocationResult, 1)
	go func() {
		defer close(done)
		stream, err := hammer.Infrav2().Boot().Wait(ctx, &infrav2.BootServiceWaitRequest{Uuid: machineID})
		if err != nil {
			result <- boundaryAllocationResult{err: err}
			return
		}
		defer func() { _ = stream.Close() }()
		if !stream.Receive() {
			result <- boundaryAllocationResult{err: fmt.Errorf("allocation stream ended: %v", stream.Err())}
			return
		}
		result <- boundaryAllocationResult{allocation: stream.Msg().Allocation}
	}()
	t.Cleanup(func() { cancel(); <-done })
	return result
}

type boundaryBMCResult struct {
	command apiv2.MachineBMCCommand
	err     error
}

func acknowledgeBoundaryBMC(t *testing.T, infra client.Client, partition, machineID string) <-chan boundaryBMCResult {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	results := make(chan boundaryBMCResult, 3)
	go func() {
		defer close(done)
		stream, err := infra.Infrav2().BMC().WaitForBMCCommand(ctx, &infrav2.WaitForBMCCommandRequest{Partition: partition})
		if err != nil {
			results <- boundaryBMCResult{err: err}
			return
		}
		defer func() { _ = stream.Close() }()
		for _, want := range []apiv2.MachineBMCCommand{
			apiv2.MachineBMCCommand_MACHINE_BMC_COMMAND_MACHINE_CREATED,
			apiv2.MachineBMCCommand_MACHINE_BMC_COMMAND_MACHINE_DELETED,
		} {
			if !stream.Receive() {
				results <- boundaryBMCResult{err: fmt.Errorf("BMC stream ended: %v", stream.Err())}
				return
			}
			msg := stream.Msg()
			if msg.Uuid != machineID || msg.BmcCommand != want || msg.CommandId == "" {
				results <- boundaryBMCResult{err: fmt.Errorf("unexpected BMC command %s for %s (id %q)", msg.BmcCommand, msg.Uuid, msg.CommandId)}
				return
			}
			_, err = infra.Infrav2().BMC().BMCCommandDone(ctx, &infrav2.BMCCommandDoneRequest{CommandId: msg.CommandId})
			results <- boundaryBMCResult{command: msg.BmcCommand, err: err}
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
	return results
}

func requireBoundaryCommand(t *testing.T, results <-chan boundaryBMCResult, want apiv2.MachineBMCCommand) {
	t.Helper()
	select {
	case result := <-results:
		require.NoError(t, result.err)
		require.Equal(t, want, result.command)
	case <-time.After(10 * time.Second):
		t.Fatalf("BMC did not acknowledge %s", want)
	}
}
