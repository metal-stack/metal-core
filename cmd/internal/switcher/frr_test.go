package switcher

import (
	"path"
	"testing"
)

func TestFrrRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			r := newFrrRenderer(c.FrrTplFile)
			testRenderer(t, r, &c, path.Join("test_data", tt, "frr.conf"))
		})
	}
}
