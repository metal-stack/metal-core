//+build mage

package main

import (
	"github.com/magefile/mage/sh"
)

// Creates the binary in the current directory. It will overwrite any existing binary.
func Build() error {
	return sh.Run("go", "build", "-o", "bin/metalcore")
}
