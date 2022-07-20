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
			a := NewFrrApplier(&c, "")
			testApplier(t, a, path.Join("test_data", tt, "frr.conf"))
		})
	}
}

func TestCustomFrrTpl(t *testing.T) {
	t.Run("customtpl", func(t *testing.T) {
		c := readConf(t, "test_data/dev/conf.yaml")
		c.FillRouteMapsAndIPPrefixLists()
		a := NewFrrApplier(&c, "test_data/dev/customtpl/frr.tpl")
		testApplier(t, a, "test_data/dev/customtpl/frr.conf")
	})
}
