//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type BUILD mg.Namespace

// Same as build:bin
func Build() error {
	return sh.RunV("go", "build", "-o", "bin/metal-core")
}

// (Re)build metal-core binary in the bin subdirectory
func (BUILD) Bin() error {
	return Build()
}

// (Re)build metal-core image
func (b BUILD) Core() error {
	return sh.RunV("docker-compose", "build", "metal-core")
}

// (Re)build metal-hammer image
func (b BUILD) Hammer() error {
	return sh.RunV("docker-compose", "build", "metal-hammer")
}

// (Re)build metal-api image
func (b BUILD) Api() error {
	return sh.RunV("docker-compose", "build", "metal-api")
}

// (Re)build all metal images
func (b BUILD) All() error {
	if err := b.Core(); err != nil {
		return err
	}
	if err := b.Hammer(); err != nil {
		return err
	}
	return b.Api()
}
