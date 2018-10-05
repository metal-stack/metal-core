//+build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type TEST mg.Namespace

// Run all tests
func Test() error {
	return runTests(func(dir string) bool {
		return true
	})
}

// Run all unit tests
func (TEST) Unit() error {
	return runTests(func(dir string) bool {
		return dir != "./tests"
	})
}

// Run all integration tests
func (TEST) Int() error {
	return runTests(func(dir string) bool {
		return dir == "./tests"
	})
}

func runTests(filter func(dir string) bool) error {
	cnt := 0
	for _, pkg := range fetchGoPackages() {
		if containsGoTests(pkg) && filter(pkg) {
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
	out, err := sh.Output("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*_test.go")
	return err == nil && len(out) > 0
}
