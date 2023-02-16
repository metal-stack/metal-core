package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/vishvananda/netlink"
)

func (a *Applier) addInterfaceToVlan(interfaceName, vlan string) error {
	a.log.Infof("add interface %s to vlan %s", interfaceName, vlan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.removeInterfaceFromVrf(ctx, interfaceName)
	if err != nil {
		// Wrapped inside the called func
		return err
	}
	err = a.c.setVlanMember(ctx, interfaceName, vlan)
	if err != nil {
		// Wrapped inside the called func
		return err
	}

	return nil
}

// removeInterfaceFromVlan removes the interface from a vlan, if the interface not bound to a vlan no op is executed,
// otherwise netlink and configdb are modified to remove the interface from a vlan.
func (a *Applier) removeInterfaceFromVlan(ctx context.Context, interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("unable to get kernel info of interface:%s %w", interfaceName, err)
	}

	err = retry.Do(
		func() error {
			vlansByInterface, err := netlink.BridgeVlanList()
			if err != nil {
				return fmt.Errorf("unable to get kernel info of vlans %w", err)
			}
			vlans, ok := vlansByInterface[int32(link.Attrs().Index)]
			if !ok {
				return nil
			}

			for _, vlan := range vlans {
				// remove from configdb
				err := a.c.deleteVlanMember(ctx, interfaceName, vlan.Vid)
				if err != nil {
					return fmt.Errorf("unable to remove vlan %d from configdb %s %w", vlan.Vid, interfaceName, err)
				}

				// remove with netlink
				// if interface is not in a vlan anymore, removing does not return with an error
				err = netlink.BridgeVlanDel(link, vlan.Vid, false, true, false, false)
				if err != nil {
					return fmt.Errorf("unable to remove vlan %d from interface %s %w", vlan.Vid, interfaceName, err)
				}
			}

			// TODO also check if interface is not configured to any vlan in configdb.
			return nil
		},
	)
	return err
}
