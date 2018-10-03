//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Env mg.Namespace

// Start a test environment
func (e Env) Up() error {
	e.Down()
	return sh.Run("docker-compose", "up")
}

// Create metal-core image and starts a test environment
func (e Env) Create() error {
	build := Build{}
	if err := build.Binary(); err != nil {
		return err
	}
	if err := sh.Run("docker-compose", "build"); err != nil {
		return err
	}
	return e.Up()
}

// Create all metal images and starts a test environment
func (e Env) CreateAll() error {
	if err := sh.Run("docker", "build", "-t", "registry.fi-ts.io/metal/discover", "../discover"); err != nil {
		return err
	}
	//if err := sh.Run("docker-compose", "-f", "../maas-service/docker-compose.yml", "build"); err != nil {
	//	return err
	//}
	return e.Create()
}

// Shut down test environment
func (Env) Down() error {
	return sh.Run("docker-compose", "down")
}
