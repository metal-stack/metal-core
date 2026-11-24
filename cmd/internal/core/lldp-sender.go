package core

import (
	"context"
	"os"
	"time"

	"github.com/metal-stack/go-lldpd/pkg/lldp"
)

// lldpSystemDescription is announced as LLDP system description.
const lldpSystemDescription = "metal-stack switch (LLDP sent by metal-core)"

// ConstantlySendLLDP sends LLDP on all switch ports with what metal-hammer
// expects from a switch, for NOSes without a suitable LLDP agent (BISDN
// Linux has none):
//
//   - chassis ID: the port's MAC address
//   - port ID: locally assigned, the port's name (the SwitchNic identifier)
//   - port description: the port's name
//   - system name: the hostname, by which metal-apiserver finds the switch
//
// Ports can appear later or be recreated (the ASIC driver creates them), so
// the port list is rescanned every interval and a sender is started for each
// new port.
func (c *Core) ConstantlySendLLDP(ctx context.Context, interval time.Duration) {
	host, err := os.Hostname()
	if err != nil {
		c.log.Error("lldp sender: unable to get hostname, not sending LLDP", "error", err)
		return
	}

	started := map[string]int{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		c.startLLDPSenders(ctx, host, interval, started)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Core) startLLDPSenders(ctx context.Context, host string, interval time.Duration, started map[string]int) {
	ports, err := c.nos.GetSwitchPorts(ctx)
	if err != nil {
		c.log.Error("lldp sender: unable to get switch ports", "error", err)
		return
	}
	for _, port := range ports {
		if len(port.HardwareAddr) == 0 || started[port.Name] == port.Index {
			continue
		}
		d, err := lldp.NewDaemon(c.log, host, lldpSystemDescription, port.Name, interval, lldp.WithPortIDSubtype(lldp.PortIDSubtypeLocal))
		if err != nil {
			c.log.Error("lldp sender: unable to start", "port", port.Name, "error", err)
			continue
		}
		d.Start()
		started[port.Name] = port.Index
		c.log.Info("lldp sender started", "port", port.Name)
	}
}
