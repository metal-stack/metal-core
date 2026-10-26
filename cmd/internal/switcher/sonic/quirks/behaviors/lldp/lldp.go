package lldp

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

const (
	lldpTable     = "LLDP"
	lldpPortTable = "LLDP_PORT"
	txInterval    = "10"
)

type LLDPConfiguration interface {
	// Configure sets the global LLDP tx_interval and the LLDP config of all managed ports
	Configure(ctx context.Context, c ConfigDB, ports []*db.Port) []error
}

// ConfigDB is the part of the CONFIG_DB the LLDP configuration needs, implemented by *db.ConfigDB
type ConfigDB interface {
	EnsureEntry(ctx context.Context, e db.Entry) error
	DeleteEntry(ctx context.Context, key db.Key) (bool, error)
}

// Configure LLDP like Community Upstream As far as we can tell this is the
// default mode unless changed by the SONiC distribution vendor and was observed
// in:
// - Edgecore SONiC (202211 based)
// - Broadcom Enterprise SONiC (202211 based)
// - Community SONiC (at least everything from 202211 until 2026xx)
// TODO: Port mode (tx-rx) is set but the port ID is not configured. Default works on metal-core on the mentioned distros.
type Community struct{}

func (Community) Configure(ctx context.Context, c ConfigDB, ports []*db.Port) []error {
	var errs []error

	err := c.EnsureEntry(ctx, db.Entry{Key: db.Key{lldpTable, "GLOBAL"}, Val: db.Val{"hello_time": txInterval}})
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to update LLDP tx_interval: %w", err))
	}
	for _, port := range ports {
		// The lldpmgrd in upstream sonic is great
		// docker exec -it lldp cat /usr/bin/lldpmgrd if you want to take a look
		//
		// CONFIG_DB Example:
		//
		//	"LLDP_PORT|Ethernet1":	{
		//	  enabled: "true", # options: true and literally anything else
		//	  mode: "RECEIVE", # options: RECEIVE and literally anything else
		//	}
		//
		// enabled == "true" and mode unset is a KeyError in lldpmgrd
		// enabled == "true" means mode == "RECEIVE" means rx-only
		// enabled == "true" means mode != "RECEIVE" means tx-only (???)
		// LLDP_PORT|EthernetX KEY missing means tx-and-rx LLDP (???)
		// enabled != "true" means means disabled
		// enabled unset means "whatever the previous state was; Pray you never have to restart" (???)
		//
		// Thankfully we only have to delete the key to make it work for metal-core
		_, err := c.DeleteEntry(ctx, db.Key{lldpPortTable, port.Name})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update LLDP settings for port %s: %w", port.Name, err))
		}
	}

	return errs
}

// Configure LLDP AsterNOS v6.1 style
type AsterNOS struct{}

func (AsterNOS) Configure(ctx context.Context, c ConfigDB, ports []*db.Port) []error {
	var errs []error

	err := c.EnsureEntry(ctx, db.Entry{Key: db.Key{lldpTable, "lldp"}, Val: db.Val{"tx_interval": txInterval}})
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to update LLDP tx_interval: %w", err))
	}
	for _, port := range ports {
		err := c.EnsureEntry(ctx, db.Entry{Key: db.Key{lldpTable, port.Name}, Val: db.Val{"lldp_local_portid": port.Alias}})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update LLDP settings for port %s: %w", port.Name, err))
		}
	}

	return errs
}
