package portstate

import (
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

type PortState interface {
	// NewChecker returns how to tell whether a port is still bridged or routed
	NewChecker(log *slog.Logger, sonicDb *db.DB) PortStateChecker
}

type AsicDb struct{}

func (AsicDb) NewChecker(log *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &sonicPortState{db: sonicDb, log: log}
}

type ConfigDb struct{}

// NewChecker checks the CONFIG_DB only. It is meant for tests that run without syncd
// filling the ASIC_DB or STATE_DB.
func (ConfigDb) NewChecker(_ *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &configDBPortState{db: sonicDb}
}
