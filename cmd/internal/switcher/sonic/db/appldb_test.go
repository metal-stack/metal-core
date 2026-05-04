package db

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestApplDB_ExistPortInitDone(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want bool
	}{
		{
			name: "not exists",
			data: test.StringMap{
				"PORT_TABLE": test.StringMap{
					"PortConfigDone": test.StringMap{
						"count": "65",
					},
				},
			},
			want: false,
		},
		{
			name: "exists",
			data: test.StringMap{
				"PORT_TABLE": test.StringMap{
					"PortInitDone": test.StringMap{
						"lanes": "0",
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &ApplDB{
				c: c,
			}
			got, err := d.ExistPortInitDone(ctx)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ApplDB.ExistPortInitDone() = %v, want %v", got, tt.want)
			}
		})
	}
}
