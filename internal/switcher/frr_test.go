package switcher

import (
	"path"
	"testing"
)

func TestFrrApplier(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			a := NewFrrApplier(&c)
			testApplier(t, a, path.Join("test_data", tt, "frr.conf"))
		})
	}
}
