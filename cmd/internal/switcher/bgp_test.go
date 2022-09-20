package switcher

import (
	"path"
	"testing"
)

func TestBgpdRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			c.FillRouteMapsAndIPPrefixLists()
			tpl := mustParseFS(bgpdTpl)
			testTemplate(t, tpl, &c, path.Join("test_data", tt, "bgpd.conf"))
		})
	}
}

func TestStaticdRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			tpl := mustParseFS(staticdTpl)
			testTemplate(t, tpl, &c, path.Join("test_data", tt, "staticd.conf"))
		})
	}
}

func TestZebraRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			tpl := mustParseFS(zebraTpl)
			testTemplate(t, tpl, &c, path.Join("test_data", tt, "zebra.conf"))
		})
	}
}
