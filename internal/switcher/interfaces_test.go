package switcher

import (
	"path"
	"testing"
)

func TestInterfacesApplier(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			a := NewInterfacesApplier(&c)
			testApplier(t, a, path.Join("test_data", tt, "interfaces"))
		})
	}
}
