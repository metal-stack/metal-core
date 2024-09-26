package vlan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type reservationInput struct {
	min   uint16
	max   uint16
	taken []uint16
	n     uint16
}

func TestReserveVlanIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    reservationInput
		expected []uint16
	}{
		{
			"Reserve one without any VLAN taken",
			reservationInput{1, 1, []uint16{}, 1},
			[]uint16{1},
		},
		{
			"Reserve multiple without any VLAN taken",
			reservationInput{1, 2, []uint16{}, 2},
			[]uint16{1, 2},
		},
		{
			"Reserve one with some VLANs taken",
			reservationInput{1, 3, []uint16{2}, 1},
			[]uint16{1},
		},
		{
			"Reserve multiple with some VLANs taken",
			reservationInput{1, 5, []uint16{1, 3, 5}, 2},
			[]uint16{2, 4},
		}, {
			"Reservation exceeds max number of VLANs",
			reservationInput{1, 1, []uint16{1}, 1},
			nil,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			m := Mapping{}
			for _, t := range tt.input.taken {
				m[t] = uint32(t)
			}
			a, _ := m.reserveVlanIDs(tt.input.min, tt.input.max, tt.input.n)
			assert.Equal(t, tt.expected, a, "reservation differs (taken: %v)", tt.input.taken)
		})
	}
}
