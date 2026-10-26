package lldp

import "github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"

type LLDPConfiguration interface {
	// Location of the Key containing the global LLDP tx_inteval
	TxIntervalMode() db.LldpSetTxIntervalMode
	// Configure the LLDP Port config for a managed port
	PortConfigMode() db.LldpSetPortMode
}

// Configure LLDP like Community Upstream As far as we can tell this is the
// default mode unless changed by the SONiC distribution vendor and was observed
// in:
// - Edgecore SONiC (202211 based)
// - Broadcom Enterprise SONiC (202211 based)
// - Community SONiC (at least everything from 202211 until 2026xx)
// TODO: Port mode (tx-rx) is set but the port ID is not configured. Default works on metal-core on the mentioned distros.
type Community struct{}

func (Community) TxIntervalMode() db.LldpSetTxIntervalMode {
	return db.CommunityIntervalMode
}

func (Community) PortConfigMode() db.LldpSetPortMode {
	return db.CommunityPortMode
}

// Configure LLDP AsterNOS v6.1 style
type AsterNOS struct{}

func (AsterNOS) TxIntervalMode() db.LldpSetTxIntervalMode {
	return db.AsterNOSIntervalMode
}

func (AsterNOS) PortConfigMode() db.LldpSetPortMode {
	return db.AsterNOSPortMode
}
