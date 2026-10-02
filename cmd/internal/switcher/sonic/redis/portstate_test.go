package redis

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

func TestNewPortStateChecker(t *testing.T) {
	tests := []struct {
		name    string
		source  PortStateSource
		want    portStateChecker
		wantErr bool
	}{
		{
			name:   "asic_db",
			source: PortStateSourceAsicDB,
			want:   &asicDBPortState{},
		},
		{
			name:   "state_db",
			source: PortStateSourceStateDB,
			want:   &stateDBPortState{},
		},
		{
			name:    "empty",
			source:  "",
			wantErr: true,
		},
		{
			name:    "unknown",
			source:  "appl_db",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newPortStateChecker(slog.Default(), &db.DB{}, tt.source)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tt.want, got)
		})
	}
}
