//+build mage

package main

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/version"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"os"
	"os/exec"
)

type BUILD mg.Namespace

// Format source code
func Fmt() {
	os.Setenv("GO111MODULE", "off")
	defer os.Setenv("GO111MODULE", "on")
	sh.RunV("go", "fmt", "./...")
}

type TEST mg.Namespace

// Run all unit and integration tests
func Test() {
	t := TEST{}
	t.Unit()
	t.Int()
}

// Run all unit tests
func (TEST) Unit() {
	os.Setenv(zapup.KeyLogLevel, "panic")
	defer os.Unsetenv(zapup.KeyLogLevel)
	if err := sh.RunV("go", "test", "-cover", "-count", "1", "-v", "./cmd/metal-core/internal/..."); err != nil {
		panic(err)
	}
}

// Run all integration tests
func (TEST) Int() {
	if err := sh.RunV("go", "test", "-cover", "-count", "1", "-v", "./cmd/metal-core/test/..."); err != nil {
		panic(err)
	}
}

// (Re)build model and metal-core binary (useful right after git clone)
func Build() {
	b := BUILD{}
	b.Model()
	b.Bin()
}

// (Re)build model
func (BUILD) Model()  {
	swagger := "swagger"
	if _, err := exec.Command("which", swagger).CombinedOutput(); err != nil {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		swagger = fmt.Sprintf("%v/bin/swagger", wd)
		if _, err := os.Stat(swagger); os.IsNotExist(err) {
			os.Mkdir("bin", 0755)
			if err := exec.Command("wget", "-O", swagger,
				"https://github.com/go-swagger/go-swagger/releases/download/v0.17.2/swagger_linux_amd64").Run(); err != nil {
				panic(err)
			}
			if err := os.Chmod(swagger, 0755); err != nil {
				panic(err)
			}
		}
	}
	os.Setenv("GO111MODULE", "off")
	defer os.Setenv("GO111MODULE", "on")
	_ = os.RemoveAll("client")
	_ = os.RemoveAll("models")
	if err := sh.RunV(swagger, "generate", "client", "-f", "domain/metal-api.json", "--skip-validation"); err != nil {
		panic(err)
	}
}

// (Re)build metal-core binary in the bin subdirectory
func (BUILD) Bin() {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("GOOS", "linux")
	gitVersion, _ := sh.Output("git", "describe", "--long", "--all")
	gitsha, _ := sh.Output("git", "rev-parse", "--short=8", "HEAD")
	buildDate, _ := sh.Output("date", "-Iseconds")
	ldflags := fmt.Sprintf("-X 'git.f-i-ts.de/cloud-native/metallib/version.Version=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Revision=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Gitsha1=%v' -X 'git.f-i-ts.de/cloud-native/metallib/version.Builddate=%v'", version.V, gitVersion, gitsha, buildDate)
	defer os.Chdir("../..")
	os.Chdir("cmd/metal-core")
	if err := sh.RunV("go", "build", "-tags", "netgo", "-ldflags", ldflags, "-o", "../../bin/metal-core"); err != nil {
		panic(err)
	}
}

// (Re)build model, metal-core binary and metal-core image
func (b BUILD) Core() {
	Build()
	if err := sh.RunV("docker-compose", "build", "metal-core"); err != nil {
		panic(err)
	}
}

// (Re)build metal-core specification
func (b BUILD) Spec() {
	b.Bin()
	_ = os.Remove("spec/metal-core.json")
	if err := sh.RunV("bin/metal-core", "spec", "spec/metal-core.json"); err != nil {
		panic(err)
	}
}
