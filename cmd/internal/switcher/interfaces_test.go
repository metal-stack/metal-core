package switcher

import (
	"path"
	"testing"
)

func TestInterfacesRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			r := newInterfacesRenderer("")
			testRenderer(t, r, &c, path.Join("test_data", tt, "interfaces"))
		})
	}
}

func TestCustomInterfacesTpl(t *testing.T) {
	t.Run("customtpl", func(t *testing.T) {
		c := readConf(t, "test_data/dev/conf.yaml")
		r := newInterfacesRenderer("test_data/dev/customtpl/interfaces.tpl")
		testRenderer(t, r, &c, "test_data/dev/customtpl/interfaces")
	})
}
