package portstate

import (
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// NewChecker returns the checker that waits for the STATE_DB, the ASIC_DB and the kernel.
func NewChecker(log *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &sonicPortState{db: sonicDb, log: log}
}
