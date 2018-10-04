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
	return sh.Run("docker-compose", "up")
}

// Start a test environment
func (e ENV) Up() error {
	return Env()
}

// Create metal-core image and start a test environment
func (e ENV) Create() error {
	if err := Build(); err != nil {
		return err
	}
	if err := sh.Run("docker-compose", "build"); err != nil {
		return err
	}
	return e.Up()
}

// Create all metal images and start a test environment
func (e ENV) Create_All() error {
	if err := sh.Run("docker", "build", "-t", "registry.fi-ts.io/metal/discover", "../discover"); err != nil {
		return err
	}
	//if err := sh.Run("docker-compose", "-f", "../maas-service/docker-compose.yml", "build"); err != nil {
	//	return err
	//}
	return e.Create()
}

// Shut down test environment
func (ENV) Down() {
	down()
}

func down() {
	sh.Run("docker-compose", "down")
}
