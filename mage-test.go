//+build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/sh"
	"os/exec"
)

// Run all tests
func Test() error {
	Fmt()
	errorCnt := 0
	for _, pkg := range fetchGoPackages() {
		if containsGoTests(pkg) {
			if err := sh.Run("go", "test", pkg); err != nil {
				errorCnt++
			}
		}
	}
	if errorCnt > 0 {
		return errors.New(fmt.Sprintf("%d test(s) failed", errorCnt))
	} else {
		return nil
	}
}

func containsGoTests(dir string) bool {
	findGoTests := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*_test.go")
	out, err := findGoTests.CombinedOutput()
	return err == nil && len(out) > 0
}
