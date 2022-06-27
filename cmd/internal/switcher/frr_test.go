package switcher

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
			testRenderer(t, tpl, &c, path.Join("test_data", tt, "frr.conf"))
		})
	}
}
