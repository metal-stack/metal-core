//+build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/sh"
)

// Run all tests
func Test() error {
	cnt := 0
	for _, pkg := range fetchGoPackages() {
		if containsGoTests(pkg) {
			if err := sh.Run("go", "test", "-v", pkg); err != nil {
				cnt++
			}
		}
	}
	if cnt > 0 {
		return errors.New(fmt.Sprintf("%d test(s) failed", cnt))
	} else {
		return nil
	}
}

func containsGoTests(dir string) bool {
	out, err := sh.Output("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*_test.go")
	return err == nil && len(out) > 0
}
