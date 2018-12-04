//+build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
	"os/exec"
)

const version = "devel"

type BUILD mg.Namespace

// (Re)build model and metal-core binary (useful right after git clone)
func Build() error {
	b := BUILD{}
	if err := b.Model(); err != nil {
		return err
	}
	return b.Bin()
}

// (Re)build model
func (BUILD) Model() error {
	swagger := "swagger"
	if _, err := exec.Command("which", swagger).CombinedOutput(); err != nil {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		swagger = fmt.Sprintf("%v/bin/swagger", wd)
		if _, err := os.Stat(swagger); os.IsNotExist(err) {
			os.Mkdir("bin", 0755)
			if err := exec.Command("wget", "-O", swagger,
				"https://github.com/go-swagger/go-swagger/releases/download/v0.17.2/swagger_linux_amd64").Run(); err != nil {
				return err
			}
			if err := os.Chmod(swagger, 0755); err != nil {
				return err
			}
		}
	}
	defer os.Setenv("GO111MODULE", "on")
	os.Setenv("GO111MODULE", "off")
	return sh.RunV("bin/swagger", "generate", "client", "-f", "domain/metal-api.json", "--skip-validation")
}

// (Re)build metal-core binary in the bin subdirectory
func (BUILD) Bin() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOOS", "linux")
	gitVersion, _ := sh.Output("git", "describe", "--long", "--all")
	gitsha, _ := sh.Output("git", "rev-parse", "--short=8", "HEAD")
	buildDate, _ := sh.Output("date", "-Iseconds")
	ldflags := fmt.Sprintf("-X 'git.f-i-ts.de/cloud-native/metallib/version.Version=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Revision=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Gitsha1=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Builddate=%v'", version, gitVersion, gitsha, buildDate)
	defer os.Chdir("../..")
	os.Chdir("cmd/metal-core")
	return sh.RunV("go", "build", "-tags", "netgo", "-ldflags", ldflags, "-o", "../../bin/metal-core")
}

// (Re)build model, metal-core binary and metal-core image
func (b BUILD) Core() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "build", "metal-core")
}

// (Re)build metal-core specification
func (b BUILD) Spec() error {
	if err := b.Bin(); err != nil {
		return err
	}
	return sh.RunV("bin/metal-core", "spec", "spec/metal-core.json")
}
