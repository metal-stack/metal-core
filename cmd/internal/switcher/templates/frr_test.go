package templates

import (
	"path"
	"testing"
)

func TestFrrTpl(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			tpl := mustParseFS(frrTpl)
			testTemplate(t, tpl, &c, path.Join("test_data", tt, "frr.conf"))
		})
	}
}

func TestCustomFrrTpl(t *testing.T) {
	c := readConf(t, "test_data/dev/conf.yaml")
	c.FillRouteMapsAndIPPrefixLists()
	a := NewFrrApplier("test_data/dev/customtpl/frr.tpl")
	testTemplate(t, a.tpl, &c, "test_data/dev/customtpl/frr.conf")
}
