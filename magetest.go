//+build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os/exec"
	"strings"
)

type TEST mg.Namespace

// Same as test:all
func Test() error {
	return runTests(func(dir string) bool {
		return true
	})
}

// Run all tests
func (TEST) All() error {
	return Test()
}

// Run all unit tests
func (TEST) Unit() error {
	return runTests(func(dir string) bool {
		return !strings.HasPrefix(dir, "./test")
	})
}

// Run all integration tests
func (TEST) Int() error {
	return runTests(func(dir string) bool {
		return dir == "./test/int"
	})
}

// Run all end-to-end tests
func (TEST) E2e() error {
	return runTests(func(dir string) bool {
		return dir == "./test/e2e"
	})
}

func runTests(filter func(dir string) bool) error {
	cnt := 0
	for _, pkg := range fetchGoPackages() {
		if !strings.HasPrefix(pkg, "./metallib") && filter(pkg) && containsGoTests(pkg) {
			if err := sh.RunV("go", "test", "-count", "1", "-v", pkg); err != nil {
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
	out, err := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*_test.go").CombinedOutput()
	return err == nil && len(out) > 0
}
