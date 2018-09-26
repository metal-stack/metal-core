//+build mage

package main

import (
	"github.com/magefile/mage/sh"
)

// Starts a test environment.
func Up() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.Run("docker-compose", "up")
}

