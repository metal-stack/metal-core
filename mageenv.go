//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type ENV mg.Namespace

// Same as env:up
func Env() error {
	down()
	return sh.RunV("docker-compose", "up")
}

// Start a test environment
func (ENV) Up() error {
	return Env()
}

// Shut down test environment
func (ENV) Down() {
	down()
}

func down() {
	sh.RunV("docker-compose", "down")
}
