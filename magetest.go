//+build mage

package main

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
)

type TEST mg.Namespace

// Run all unit and integration tests
func Test() {
	t := TEST{}
	t.Unit()
	t.Int()
}

// Run all unit tests
func (TEST) Unit() {
	os.Setenv(zapup.KeyLogLevel, "panic")
	defer os.Unsetenv(zapup.KeyLogLevel)
	if err := sh.RunV("go", "test", "-cover", "-count", "1", "-v", "./cmd/metal-core/internal/..."); err != nil {
		panic(err)
	}
}

// Run all integration tests
func (TEST) Int() {
	if err := sh.RunV("go", "test", "-cover", "-count", "1", "-v", "./cmd/metal-core/test/..."); err != nil {
		panic(err)
	}
}
