//+build mage

package main

import (
	"github.com/magefile/mage/sh"
)

// Creates the binary metalcore in the bin subdirectory. It will overwrite any existing binary.
func Build() error {
	return sh.Run("go", "build", "-o", "bin/metalcore")
}

// Creates the docker image metalcore that runs the binary metalcore by default.
func Docker() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.Run("docker", "build", "-t", "metalcore", ".")
}
