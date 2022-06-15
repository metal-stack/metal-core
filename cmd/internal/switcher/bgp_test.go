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
			r := newBgpdRenderer()
			testRenderer(t, r, &c, path.Join("test_data", tt, "bgpd.conf"))
		})
	}
}

func TestStaticdRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			r := newStaticdRenderer()
			testRenderer(t, r, &c, path.Join("test_data", tt, "staticd.conf"))
		})
	}
}

func TestZebraRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			r := newZebraRenderer()
			testRenderer(t, r, &c, path.Join("test_data", tt, "zebra.conf"))
		})
	}
}
