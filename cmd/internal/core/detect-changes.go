package core

import (
	"context"
	"net"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
)

const (
	detectChangesInterval = time.Minute
)

func (c *Core) DetectInterfaceChanges(ctx context.Context, discoveryResultChan chan lldp.DiscoveryResult) {
	ifaceTicker := time.NewTicker(detectChangesInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ifaceTicker.C:
			c.log.Info("checking for port changes")
			ifs, err := net.Interfaces()
			if err != nil {
				c.log.Errorw("unable to gather interfaces, ignoring", "error", err)
				continue
			}
			actualInterfaces := []string{}
			for _, iface := range ifs {
				actualInterfaces = append(actualInterfaces, iface.Name)
			}
			existingInterfaces := []string{}
			c.interfaces.Range(func(key, value any) bool {
				existingInterfaces = append(existingInterfaces, key.(string))
				return true
			})

			addedInterfaces, removedInterfaces := difference(existingInterfaces, actualInterfaces)

			if len(addedInterfaces) == 0 && len(removedInterfaces) == 0 {
				c.log.Infow("switch interfaces changed, re-register switch")
				c.RegisterSwitch()
			} else {
				c.log.Info("no port changes detected")
				continue
			}

			for _, i := range removedInterfaces {
				c.log.Infow("remove lldp discovery for", "interfaces", i)
				c.stopLLDPDiscovery(i)
			}
			for _, i := range addedInterfaces {
				iface, err := net.InterfaceByName(i)
				if err != nil {
					c.log.Errorw("unable to get interface by name", "interface", i, "error", err)
					continue
				}
				c.log.Infow("add lldp discovery for", "interfaces", *iface)
				c.startLLDPDiscovery(ctx, discoveryResultChan, *iface)
			}
		}
	}
}
