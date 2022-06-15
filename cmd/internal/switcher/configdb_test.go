package switcher

import (
	"path"
	"testing"
)

func TestConfigdbRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			r := configdbRenderer{}
			testRenderer(t, &r, &c, path.Join("test_data", tt, "configdb.json"))
		})
	}
}
