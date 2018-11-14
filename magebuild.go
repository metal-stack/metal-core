//+build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"io/ioutil"
)

const version = "devel"

type BUILD mg.Namespace

// Same as build:bin
func Build() error {
	defer os.Setenv("CGO_ENABLED", "1")
	prepareEnv()
	gitVersion, _ := sh.Output("git", "describe", "--long", "--all")
	gitsha, _ := sh.Output("git", "rev-parse", "--short=8", "HEAD")
	buildDate, _ := sh.Output("date", "-Iseconds")
	ldflags := fmt.Sprintf("-X 'git.f-i-ts.de/cloud-native/metallib/version.Version=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Revision=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Gitsha1=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Builddate=%v'", version, gitVersion, gitsha, buildDate)
	defer os.Chdir("../..")
	os.Chdir("cmd/metal-core")
	return sh.RunV("go", "build", "-tags", "netgo", "-ldflags", ldflags, "-o", "../../bin/metal-core")
}

// (Re)build metal-core specification
func (b BUILD) Spec() error {
	defer exec.Command("docker", "rm", "-f", "spec-metal-core").Run()
	exec.Command("docker", "rm", "-f", "spec-metal-core").Run()
	if err := b.Core(); err != nil {
		return err
	} else if err := sh.RunV("docker-compose", "run", "--name", "spec-metal-core", "-d", "metal-core"); err != nil {
		return err
	} else if out, err := sh.Output("docker", "inspect", "-f", "{{ .NetworkSettings.Networks.metal.IPAddress }}", "spec-metal-core"); err != nil {
		return err
	} else if out, err := sh.Output("curl", "-s", fmt.Sprintf("http://%v:4242/apidocs.json", out)); err != nil {
		return err
	} else {
		return ioutil.WriteFile("spec/metal-core.json", []byte(out), 0644)
	}
}

// (Re)build metal-core binary in the bin subdirectory
func (BUILD) Bin() error {
	return Build()
}

// (Re)build model
func (BUILD) Model() error {
	swagger := "swagger"
	if _, err := exec.Command("which", swagger).CombinedOutput(); err != nil {
		swagger = "bin/swagger"
		if _, err := os.Stat(swagger); os.IsNotExist(err) {
			os.Mkdir("bin", 0755)
			if err := exec.Command("wget", "-O", swagger,
				"https://github.com/go-swagger/go-swagger/releases/download/v0.17.2/swagger_linux_amd64").Run(); err != nil {
				return err
			} else if err := os.Chmod(swagger, 0755); err != nil {
				return err
			}
		}
	}
	defer os.Setenv("GO111MODULE", "on")
	os.Setenv("GO111MODULE", "off")
	return sh.RunV("bin/swagger", "generate", "client", "-f", "domain/metal-api.json", "--skip-validation")
}

// (Re)build metal-core image
func (b BUILD) Core() error {
	if err := b.Bin(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "build", "metal-core")
}

// (Re)build metal-api image
func (b BUILD) Api() error {
	//	return sh.RunV("docker-compose", "build", "metal-api")
	defer os.Chdir("../../../metal-core")
	os.Chdir("../metal-api/cmd/metal-api")
	defer os.Setenv("CGO_ENABLED", "1")
	prepareEnv()
	if err := exec.Command("go", "build", "-o", "../../bin/metal-api", ".").Run(); err != nil {
		return err
	}
	return sh.RunV("docker", "build", "-t", "registry.fi-ts.io/metal/metal-api", "-f", "../../Dockerfile.dev", "../..")
}

// (Re)build all metal images
func (b BUILD) All() error {
	if err := b.Core(); err != nil {
		return err
	}
	return b.Api()
}

func prepareEnv() {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOOS", "linux")
}
