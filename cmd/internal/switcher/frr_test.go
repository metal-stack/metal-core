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
			r := newFrrRenderer("")
			testRenderer(t, r, &c, path.Join("test_data", tt, "frr.conf"))
		})
	}
}

func TestCustomFrrTpl(t *testing.T) {
	t.Run("customtpl", func(t *testing.T) {
		c := readConf(t, "test_data/dev/conf.yaml")
		c.FillRouteMapsAndIPPrefixLists()
		r := newFrrRenderer("test_data/dev/customtpl/frr.tpl")
		testRenderer(t, r, &c, "test_data/dev/customtpl/frr.conf")
	})
}
