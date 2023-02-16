package configdb

import (
	"context"
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"

	"github.com/avast/retry-go/v4"
)

type InterfaceConfiguration struct {
	Name string
	Vlan *Vlan
	Vrf  *Vrf
}

type Vlan struct {
	Name string
}
type Vrf struct {
	Name string
}

type ConfigDB interface {
	// Upper API

	// ConfigureInterface must be called for every Interface when changes arrive from metal-api
	// It will check the current state of the interface on the kernel side and on the sonic-configdb
	// and issue the required netlink and configdb changes to bring the Interface to this desired state.
	// If desired state could not be reached an error is thrown
	ConfigureInterface(InterfaceConfiguration) error
}

type configdb struct {
	r   *database
	log *zap.SugaredLogger
}

func New(log *zap.SugaredLogger, opt *Options) ConfigDB {
	return &configdb{
		r: newRedis(opt),
		log: log,
	}
}

func (c *configdb) ConfigureInterface(config InterfaceConfiguration) error {
	if config.Vlan != nil && config.Vrf != nil {
		return fmt.Errorf("either vlan or vrf must be configured not both")
	}
	c.log.Infow("configure", "interface", config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if config.Vlan != nil {
		err := c.removeInterfaceFromVRF(ctx, config.Name)
		if err != nil {
			// Wrapped inside the called func
			return err
		}
		err = c.addInterfaceToVLAN(ctx, config.Name, config.Vlan.Name)
		if err != nil {
			// Wrapped inside the called func
			return err
		}
	}

	if config.Vrf != nil {
		err := c.removeInterfaceFromVLAN(ctx, config.Name)
		if err != nil {
			// Wrapped inside the called func
			return err
		}
		err = c.addInterfaceToVRF(ctx, config.Name, config.Vrf.Name)
		if err != nil {
			// Wrapped inside the called func
			return err
		}
	}

	return nil
}

func (c *configdb) addInterfaceToVLAN(ctx context.Context, interfaceName, vlan string) error {
	return c.r.setVLANMember(ctx, interfaceName, vlan)
}

// removeInterfaceFromVLAN removes the interface from a vlan, if the interface not bound to a vlan no op is executed,
// otherwise netlink and configdb are modified to remove the interface from a vlan.
func (c *configdb) removeInterfaceFromVLAN(ctx context.Context, interfaceName string) error {
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
				err := c.r.deleteVLANMember(ctx, interfaceName, vlan.Vid)
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

func (c *configdb) addInterfaceToVRF(ctx context.Context, interfaceName, vrf string) error {
	return c.r.setVRFMember(ctx, interfaceName, vrf)
}

func (c *configdb) removeInterfaceFromVRF(ctx context.Context, interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("unable to get kernel info of interface:%s %w", interfaceName, err)
	}

	err = retry.Do(
		func() error {
			// remove from configdb
			err := c.r.deleteVRFMember(ctx, interfaceName)
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
