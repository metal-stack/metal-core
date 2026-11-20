package portstate

import (
	"context"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// NewTestChecker returns a checker without conditions.
// It is meant for tests that only run against a CONFIG_DB, without the SONiC daemons that would apply it.
func NewTestChecker() PortStateChecker {
	return noPortState{}
}

type noPortState struct{}

func (noPortState) Refresh(context.Context) error                          { return nil }
func (noPortState) NotBridged(string) []db.Condition                       { return nil }
func (noPortState) NotRouted(string) []db.Condition                        { return nil }
func (noPortState) BridgedIn(string, string) []db.Condition                { return nil }
func (noPortState) RoutedIn(string, string) []db.Condition                 { return nil }
func (noPortState) HasAdminStatus(string, types.PortStatus) []db.Condition { return nil }
