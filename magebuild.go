//+build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const version = "devel"

type BUILD mg.Namespace

// Same as build:bin
func Build() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOOS", "linux")
	gitVersion, _ := sh.Output("git", "describe", "--long", "--all")
	gitsha, _ := sh.Output("git", "rev-parse", "--short=8", "HEAD")
	buildDate, _ := sh.Output("date", "-Iseconds")
	ldflags := fmt.Sprintf("-X 'git.f-i-ts.de/cloud-native/metallib/version.Version=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Revision=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Gitsha1=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Builddate=%v'", version, gitVersion, gitsha, buildDate)
	return sh.RunV("go", "build", "-tags", "netgo", "-ldflags", ldflags, "-o", "bin/metal-core")
}

// (Re)build metal-core binary in the bin subdirectory
func (BUILD) Bin() error {
	return Build()
}

// (Re)build model
func (BUILD) Model() error {
	if _, err := os.Stat("bin/swagger"); os.IsNotExist(err) {
		os.Mkdir("bin", 0755)
		if err := exec.Command("wget", "-O", "bin/swagger",
			"https://github.com/go-swagger/go-swagger/releases/download/v0.17.2/swagger_linux_amd64").Run(); err != nil {
			return err
		} else if err := os.Chmod("bin/swagger", 0755); err != nil {
			return err
		}
	}
	defer os.Setenv("GO111MODULE", "on")
	os.Setenv("GO111MODULE", "off")
	return sh.RunV("bin/swagger", "generate", "client", "-f", "internal/domain/metal-api.json", "--skip-validation")
}

// (Re)build metal-core image
func (b BUILD) Core() error {
	if err := b.Bin(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "build", "metal-core")
}

// (Re)build metal-hammer image
func (b BUILD) Hammer() error {
//	return sh.RunV("docker-compose", "build", "metal-hammer")
	defer os.Chdir("../metal-core")
	os.Chdir("../metal-hammer")
	return sh.RunV("docker-make", "--no-push", "--Lint")
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
