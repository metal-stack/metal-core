package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBootMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr string
	}{
		{name: "pxe", mode: "pxe"},
		{name: "l3", mode: "l3"},
		{name: "unknown mode", mode: "dhcp", wantErr: "unknown boot mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBootMode(tt.mode)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, BootMode(tt.mode), got)
		})
	}
}
