//+build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magiconair/properties"
	"io/ioutil"
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

// (Re)build metal-core specification
func (b BUILD) Spec() error {
	var env struct {
		MetalCoreIP   string `properties:"METAL_CORE_IP"`
		MetalCorePort int    `properties:"METAL_CORE_PORT"`
	}
	p := properties.MustLoadFile(".env", properties.UTF8)
	if err := p.Decode(&env); err != nil {
		return err
	}

	defer exec.Command("docker-compose", "rm", "-sf", "metal-core").Run()
	exec.Command("docker-compose", "rm", "-sf", "metal-core").Run()
	if err := b.Core(); err != nil {
		return err
	}
	if err := sh.RunV("docker-compose", "run", "--name", "metal-core", "-d", "metal-core"); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	out, err := sh.Output("curl", "-s", fmt.Sprintf("http://%v:%d/apidocs.json", env.MetalCoreIP, env.MetalCorePort))
	if err != nil {
		return err
	}
	return ioutil.WriteFile("spec/metal-core.json", []byte(out), 0644)
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

// (Re)build model, metal-core binary and metal-core image
func (b BUILD) Core() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "build", "metal-core")
}

// (Re)build metal-api image
func (b BUILD) Api() error {
	defer os.Chdir("../../../metal-core")
	os.Chdir("../metal-api/cmd/metal-api")
	defer os.Setenv("CGO_ENABLED", "1")
	prepareEnv()
	if err := exec.Command("go", "build", "-o", "../../bin/metal-api", ".").Run(); err != nil {
		return err
	}
	return sh.RunV("docker", "build", "-t", "registry.fi-ts.io/metal/metal-api", "-f", "../../Dockerfile.dev", "../..")
}

// (Re)build model and metal-core binary as well as metal-core and metal-api images
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
