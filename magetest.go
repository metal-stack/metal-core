//+build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type INT mg.Namespace

// Run all tests
func Test() error {
	return runTests(func(dir string) bool {
		return true
	})
}

// Run all unit tests
func Unit() error {
	return runTests(func(dir string) bool {
		return dir != "./tests"
	})
}

// Run all integration tests
func Int() error {
	return runTests(func(dir string) bool {
		return dir == "./tests"
	})
}

// (Re)build metal-core image and run all integration tests
func (INT) Build() error {
	b := BUILD{}
	if err := b.Image(); err != nil {
		return err
	}
	return Int()
}

// (Re)build all metal images and run all integration tests
func (INT) Scratch() error {
	b := BUILD{}
	if err := b.All(); err != nil {
		return err
	}
	return Int()
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
