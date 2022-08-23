package core

import (
	"testing"

	"golang.org/x/exp/slices"
)

func Test_difference(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name        string
		old         []string
		new         []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "equal",
			old:         []string{"a", "b", "c"},
			new:         []string{"a", "b", "c"},
			wantAdded:   []string{},
			wantRemoved: []string{},
		},
		{
			name:        "one added",
			old:         []string{"a", "b", "c"},
			new:         []string{"a", "b", "d", "c"},
			wantAdded:   []string{"d"},
			wantRemoved: []string{},
		},
		{
			name:        "one removed",
			old:         []string{"a", "b", "d", "c"},
			new:         []string{"a", "b", "c"},
			wantAdded:   []string{},
			wantRemoved: []string{"d"},
		},
		{
			name:        "more added and removed",
			old:         []string{"a", "x", "b", "d", "c"},
			new:         []string{"a", "b", "c", "z", "j"},
			wantAdded:   []string{"z", "j"},
			wantRemoved: []string{"x", "d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved := difference(tt.old, tt.new)
			if !slices.Equal(gotAdded, tt.wantAdded) {
				t.Errorf("difference() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !slices.Equal(gotRemoved, tt.wantRemoved) {
				t.Errorf("difference() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
		})
	}
}
