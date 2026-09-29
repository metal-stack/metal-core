//go:build client && boundary

package core

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/metal-stack/api/go/client"
	adminv2 "github.com/metal-stack/api/go/metalstack/admin/v2"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	infrav2 "github.com/metal-stack/api/go/metalstack/infra/v2"
	"github.com/stretchr/testify/require"
)

func testBootPrefixReuseBoundary(t *testing.T, admin, infra client.Client, artifactURL string) {
	t.Helper()
	t.Log("boot prefixes: a different switch port can reuse a released prefix")
	ctx := t.Context()
	const partition = "prefix-reuse"
	_, err := admin.Adminv2().Partition().Create(ctx, &adminv2.PartitionServiceCreateRequest{Partition: &apiv2.Partition{
		Id: partition, BootConfiguration: &apiv2.PartitionBootConfiguration{ImageUrl: artifactURL, KernelUrl: artifactURL},
	}})
	require.NoError(t, err)
	// Exactly two /64s make reuse observable without requiring any allocation order.
	_, err = admin.Adminv2().Network().Create(ctx, &adminv2.NetworkServiceCreateRequest{
		Id: new("prefix-reuse-boot"), Partition: new(partition), Type: apiv2.NetworkType_NETWORK_TYPE_BOOT,
		Prefixes: []string{"fd00:21::/63"}, Vrf: new(uint32(43)),
		DefaultChildPrefixLength: &apiv2.ChildPrefixLength{Ipv6: new(uint32(64))},
	})
	require.NoError(t, err)
	register := func(id string, ports ...string) (*infrav2.SwitchServiceRegisterResponse, error) {
		var nics []*apiv2.SwitchNic
		for _, port := range ports {
			nics = append(nics, &apiv2.SwitchNic{Name: port, Identifier: port, State: &apiv2.NicState{Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP}})
		}
		return infra.Infrav2().Switch().Register(ctx, &infrav2.SwitchServiceRegisterRequest{Switch: &apiv2.Switch{
			Id: id, Partition: partition, Rack: new("reuse-rack"), ManagementIp: "192.0.2.4", Nics: nics,
			Os: &apiv2.SwitchOS{Vendor: apiv2.SwitchOSVendor_SWITCH_OS_VENDOR_SONIC, Version: "202411", MetalCoreVersion: "boundary-test"},
		}})
	}
	first, err := register("prefix-owner", "Ethernet0", "Ethernet4")
	require.NoError(t, err)
	require.Len(t, first.Switch.Nics, 2)
	assigned := map[string]string{}
	for _, nic := range first.Switch.Nics {
		assigned[nic.Name] = nic.GetBootPrefix()
	}
	require.NotEmpty(t, assigned["Ethernet0"])
	require.NotEmpty(t, assigned["Ethernet4"])
	require.NotEqual(t, assigned["Ethernet0"], assigned["Ethernet4"])

	_, err = register("prefix-recipient", "Ethernet8")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "live reservations must not be reused")

	// Removing Ethernet0 releases its reservation. Ethernet4 remains active.
	first, err = register("prefix-owner", "Ethernet4")
	require.NoError(t, err)
	require.Len(t, first.Switch.Nics, 1)
	require.Equal(t, assigned["Ethernet4"], first.Switch.Nics[0].GetBootPrefix())
	second, err := register("prefix-recipient", "Ethernet8")
	require.NoError(t, err)
	require.Len(t, second.Switch.Nics, 1)
	require.Equal(t, "Ethernet8", second.Switch.Nics[0].Name)
	require.Equal(t, assigned["Ethernet0"], second.Switch.Nics[0].GetBootPrefix(), "the only free prefix must be reusable by a different port on a different switch")
}
