//+build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
	"os/exec"
	"strings"
)

type TEST mg.Namespace

// Same as test:all
func Test() error {
	t := TEST{}
	if err := t.Unit(); err != nil {
		return err
	}
	return t.Int()
}

// Run all unit tests
func (TEST) Unit() error {
	defer os.Unsetenv("ZAP_LEVEL")
	os.Setenv("ZAP_LEVEL", "panic")
	return runTests(func(dir string) bool {
		return !strings.HasPrefix(dir, "./cmd/metal-core/test")
	})
}

// Run all integration tests
func (TEST) Int() error {
	return runTests(func(dir string) bool {
		return dir == "./cmd/metal-core/test"
	})
}

func runTests(filter func(dir string) bool) error {
	cnt := 0
	for _, pkg := range fetchGoPackages() {
		if filter(pkg) && containsGoTests(pkg) {
			if err := sh.RunV("go", "test", "-cover", "-count", "1", "-v", pkg); err != nil {
				cnt++
			}
		}
	}
	if cnt > 0 {
		return fmt.Errorf("%d test(s) failed", cnt)
	} else {
		return nil
	}
}

func containsGoTests(dir string) bool {
	out, err := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*_test.go").CombinedOutput()
	return err == nil && len(out) > 0
}
