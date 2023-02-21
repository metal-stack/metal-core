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
	inVrf, err := isVrfMember(interfaceName)
	if err != nil {
		return err
	}
	if !inVrf {
		return nil
	}

	err = retry.Do(
		func() error {
			// remove from configdb
			err := a.c.deleteVrfMember(ctx, interfaceName)
			if err != nil {
				return fmt.Errorf("unable to remove interface %s from a vrf from configdb: %w", interfaceName, err)
			}

			inVrf, err = isVrfMember(interfaceName)
			if err != nil {
				return err
			}
			if inVrf {
				return fmt.Errorf("interface %s is still member of a vrf", interfaceName)
			}

			return nil
		},
	)
	return err
}

func isVrfMember(interfaceName string) (bool, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return false, fmt.Errorf("unable to get kernel info of the interface %s: %w", interfaceName, err)
	}

	if link.Attrs().MasterIndex == 0 {
		return false, nil
	}

	master, err := netlink.LinkByIndex(link.Attrs().MasterIndex)
	if err != nil {
		return false, fmt.Errorf("unable to get the master of the interface %s: %w", interfaceName, err)
	}
	return master.Type() == "vrf", nil
}
