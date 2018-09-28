//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Build mg.Namespace

// Create the binary metalcore in the bin subdirectory (it will overwrite any existing binary)
func (Build) Binary() error {
	Fmt()
	return sh.Run("go", "build", "-o", "bin/metalcore")
}

// Create the docker image 'registry.fi-ts.io/metal/metalcore:latest' that runs the binary metalcore by default
func (b Build) Image() error {
	if err := b.Binary(); err != nil {
		return err
	}
	return sh.Run("docker-compose", "build")
}
