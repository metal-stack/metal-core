//+build mage

package main

import (
	"github.com/magefile/mage/sh"
)

// Starts a test environment.
func Up() error {
	build := Build{}
	if err := build.Binary(); err != nil {
		return err
	}
	return sh.Run("docker-compose", "up", "--build")
}

