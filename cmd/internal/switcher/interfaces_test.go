package switcher

import (
	"path"
	"testing"
)

func TestInterfacesApplier(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			a := NewInterfacesApplier(&c, "")
			testApplier(t, a, path.Join("test_data", tt, "interfaces"))
		})
	}
}

func TestCustomInterfacesTpl(t *testing.T) {
	t.Run("customtpl", func(t *testing.T) {
		c := readConf(t, "test_data/dev/conf.yaml")
		a := NewInterfacesApplier(&c, "test_data/dev/customtpl/interfaces.tpl")
		testApplier(t, a, "test_data/dev/customtpl/interfaces")
	})
}
