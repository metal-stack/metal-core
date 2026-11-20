package quirks

import (
	"context"
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/portstate"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// NewTestQuirks returns the community_old profile without port state checks.
// It is meant for tests that only run against a CONFIG_DB, without the SONiC daemons that would apply it.
func NewTestQuirks() SonicQuirks {
	q := QuirkProfiles["community_old"]()
	q.PortState = noPortState{}
	return q
}

type noPortState struct{}

func (noPortState) NewChecker(*slog.Logger, *db.DB) portstate.PortStateChecker {
	return noPortStateChecker{}
}

// noPortStateChecker returns no conditions, as nothing applies the CONFIG_DB that could be waited for.
type noPortStateChecker struct{}

func (noPortStateChecker) Refresh(context.Context) error                          { return nil }
func (noPortStateChecker) NotBridged(string) []db.Condition                       { return nil }
func (noPortStateChecker) NotRouted(string) []db.Condition                        { return nil }
func (noPortStateChecker) BridgedIn(string, string) []db.Condition                { return nil }
func (noPortStateChecker) RoutedIn(string, string) []db.Condition                 { return nil }
func (noPortStateChecker) HasAdminStatus(string, types.PortStatus) []db.Condition { return nil }
