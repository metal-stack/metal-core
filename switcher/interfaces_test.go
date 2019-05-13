package switcher

import (
	"fmt"
	"path"
	"testing"
)

func TestInterfacesApplier(t *testing.T) {
	for _, tc := range listTestCases() {
		t.Run(fmt.Sprintf("%s", tc), func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tc, "conf.yaml"))
			a := NewInterfacesApplier(&c)
			testApplier(t, a, path.Join("test_data", tc, "interfaces"))
		})
	}
}
