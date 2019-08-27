package switcher

import (
	"fmt"
	"path"
	"testing"
)

func TestFrrApplier(t *testing.T) {
	for _, tc := range listTestCases() {
		t.Run(fmt.Sprintf("%s", tc), func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tc, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			a := NewFrrApplier(&c)
			testApplier(t, a, path.Join("test_data", tc, "frr.conf"))
		})
	}
}
