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
			testRenderer(t, tpl, &c, path.Join("test_data", tt, "interfaces"))
		})
	}
}
