package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/vishvananda/netlink"
)

func (a *Applier) addInterfaceToVrf(interfaceName, vrf string) error {
	a.log.Infof("add interface %s to vrf %s", interfaceName, vrf)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.removeInterfaceFromVlan(ctx, interfaceName)
	if err != nil {
		// Wrapped inside the called func
		return err
	}
	err = a.c.setVrfMember(ctx, interfaceName, vrf)
	if err != nil {
		// Wrapped inside the called func
		return err
	}

	return nil
}

func (a *Applier) removeInterfaceFromVrf(ctx context.Context, interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("unable to get kernel info of interface:%s %w", interfaceName, err)
	}

	if link.Attrs().MasterIndex == 0 {
		return nil
	}

	master, err := netlink.LinkByIndex(link.Attrs().MasterIndex)
	if err != nil {
		return fmt.Errorf("unable to get the master of interface:%s %w", interfaceName, err)
	}
	if master.Type() != "vrf" {
		return nil
	}

	err = retry.Do(
		func() error {
			// remove from configdb
			err := a.c.deleteVrfMember(ctx, interfaceName)
			if err != nil {
				return fmt.Errorf("unable to remove vrf from configdb %s %w", interfaceName, err)
			}

			// remove with netlink
			// if there is a master (vrfname) remove it
			err = netlink.LinkSetNoMaster(link)
			if err != nil {
				return fmt.Errorf("unable to remove vrf from interface %s %w", interfaceName, err)
			}

			return nil
		},
	)
	return err
}
