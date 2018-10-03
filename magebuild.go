//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Build mg.Namespace

// Create the binary metal-core in the bin subdirectory (it will overwrite any existing binary)
func (Build) Binary() error {
	Fmt()
	return sh.Run("go", "build", "-o", "bin/metal-core")
}

// Create the docker image 'registry.fi-ts.io/metal/metal-core:latest' that runs the binary metal-core by default
func (b Build) Image() error {
	if err := b.Binary(); err != nil {
		return err
	}
	return sh.Run("docker-compose", "build")
}
