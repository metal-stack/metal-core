package quirks

import (
	"context"
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// NewTestQuirks returns the community profile, but reads the port state from the CONFIG_DB.
// It is meant for tests that only run against a CONFIG_DB, without syncd filling the ASIC_DB.
func NewTestQuirks() Quirks {
	return testQuirks{}
}

type testQuirks struct {
	communityQuirks
}

func (testQuirks) Name() string { return "test" }

func (testQuirks) PortState(_ *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &configDBPortState{db: sonicDb}
}

// configDBPortState treats a port as bridged as long as it is a VLAN member and as routed
// as long as it has an interface configuration in the CONFIG_DB.
type configDBPortState struct {
	db *db.DB
}

func (s *configDBPortState) Refresh(context.Context) error { return nil }

func (s *configDBPortState) IsBridged(ctx context.Context, interfaceName string) (bool, error) {
	vlans, err := s.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return false, err
	}
	return len(vlans) > 0, nil
}

func (s *configDBPortState) IsRouted(ctx context.Context, interfaceName string) (bool, error) {
	return s.db.Config.ExistInterfaceConfiguration(ctx, interfaceName)
}
