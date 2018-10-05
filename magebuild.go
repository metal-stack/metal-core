//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type BUILD mg.Namespace

// (Re)build metal-core binary in the bin subdirectory
func Build() error {
	Fmt()
	return sh.RunV("go", "build", "-o", "bin/metal-core")
}

// (Re)build metal-core image
func (BUILD) Image() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "build")
}

// (Re)build all metal images, i.e. 'registry.fi-ts.io/metal/[metal-hammer|metal-api|metal-core]:latest'
func (b BUILD) All() error {
	if err := sh.RunV("docker", "build", "-t", "registry.fi-ts.io/metal/metal-hammer", "../metal-hammer"); err != nil {
		return err
	}
	if err := sh.RunV("docker-compose", "-f", "../maas-service/docker-compose.yml", "build"); err != nil {
		return err
	}
	return b.Image()
}
