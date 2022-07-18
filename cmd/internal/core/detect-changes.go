package core

import (
	"context"
	"net"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
	"golang.org/x/exp/slices"
)

const (
	detectChangesInterval = 5 * time.Minute
)

func (c *Core) DetectInterfaceChanges(ctx context.Context, discoveryResultChan chan lldp.DiscoveryResult) {
	ifaceTicker := time.NewTicker(detectChangesInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ifaceTicker.C:
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

			if !slices.Equal(existingInterfaces, actualInterfaces) {
				c.log.Infow("switch interfaces changed, reregister switch")
				c.RegisterSwitch()
			}

			addedInterfaces, removedInterfaces := difference(existingInterfaces, actualInterfaces)
			for _, i := range removedInterfaces {
				c.log.Infow("remove lldp discovery for", "interfaces", i)
				c.stopLLDPDiscovery(i)
			}
			for _, i := range addedInterfaces {
				c.log.Infow("add lldp discovery for", "interfaces", i)
				c.startLLDPDiscovery(ctx, discoveryResultChan, i)
			}
		}
	}
}
