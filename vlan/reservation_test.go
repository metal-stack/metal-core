package vlan

import (
	"fmt"
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
	tt := []struct {
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
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			m := Mapping{}
			for _, t := range tc.input.taken {
				m[t] = uint32(t)
			}
			a, _ := m.reserveVlanIDs(tc.input.min, tc.input.max, tc.input.n)
			assert.Equal(t, tc.expected, a, fmt.Sprintf("reservation differs (taken: %v)", tc.input.taken))
		})
	}

}
