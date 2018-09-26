//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Build mg.Namespace

// Creates the binary metalcore in the bin subdirectory. It will overwrite any existing binary.
func (Build) Binary() error {
	return sh.Run("go", "build", "-o", "bin/metalcore")
}

// Creates the docker image metalcore that runs the binary metalcore by default.
func (b Build) Image() error {
	if err := b.Binary(); err != nil {
		return err
	}
	return sh.Run("docker", "build", "-t", "metalcore", ".")
}
