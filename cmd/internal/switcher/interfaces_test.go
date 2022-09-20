package switcher

import (
	"path"
	"testing"
)

func TestInterfacesTpl(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			tpl := mustParseFS(interfacesTpl)
			testTemplate(t, tpl, &c, path.Join("test_data", tt, "interfaces"))
		})
	}
}

func TestCustomInterfacesTpl(t *testing.T) {
	c := readConf(t, "test_data/dev/conf.yaml")
	a := NewInterfacesApplier("test_data/dev/customtpl/interfaces.tpl")
	testTemplate(t, a.tpl, &c, "test_data/dev/customtpl/interfaces")
}
