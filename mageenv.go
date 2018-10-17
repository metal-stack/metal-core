//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os/exec"
	"time"
)

type ENV mg.Namespace

// Same as env:up
func Env() error {
	down()
	go func() {
		time.Sleep(8 * time.Second)
		exec.Command("docker", "rm", "-f", "metal-hammer").Run()
		exec.Command("docker-compose", "up", "metal-hammer").Run()
	}()
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
