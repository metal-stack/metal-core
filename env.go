//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Env mg.Namespace

// Starts a test environment.
func (Env) Up() error {
	return sh.Run("docker-compose", "up")
}

// Creates and starts a test environment.
func (Env) Create() error {
	build := Build{}
	if err := build.Binary(); err != nil {
		return err
	}
	return sh.Run("docker-compose", "up", "--build")
}

// Shuts down test environment.
func (Env) Down() error {
	return sh.Run("docker-compose", "down")
}
