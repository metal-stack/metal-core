//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type ENV mg.Namespace

// Start a test environment
func Env() error {
	down()
	return sh.RunV("docker-compose", "up")
}

// (Re)build metal-core image and start a test environment
func (ENV) Build() error {
	b := BUILD{}
	if err := b.Image(); err != nil {
		return err
	}
	return Env()
}

// (Re)build all metal images and start a test environment
func (ENV) Scratch() error {
	b := BUILD{}
	if err := b.All(); err != nil {
		return err
	}
	return Env()
}

// Shut down test environment
func (ENV) Down() {
	down()
}

func down() {
	sh.RunV("docker-compose", "down")
}
