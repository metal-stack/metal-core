package switcher

import (
	"path"
	"testing"
)

//nolint:unused,deadcode
func aTestFrrApplier(t *testing.T) { //TODO
	for _, tc := range listTestCases() {
		t.Run(tc, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tc, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			a := NewFrrApplier(&c)
			testApplier(t, a, path.Join("test_data", tc, "frr.conf"))
		})
	}
}
