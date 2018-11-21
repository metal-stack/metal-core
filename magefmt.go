//+build mage

package main

import (
	"os"
	"os/exec"
	"strings"
)

// Format source code
func Fmt() {
	defer os.Setenv("GO111MODULE", "on")
	os.Setenv("GO111MODULE", "off")
	for _, pkg := range fetchGoPackages() {
		if containsGoSources(pkg) {
			exec.Command("go", "fmt", pkg).Run()
		}
	}
}

func fetchGoPackages() []string {
	pp := []string{}
	if out, err := exec.Command("find", ".", "-mindepth", "1", "-type", "d").CombinedOutput(); err == nil && len(out) > 0 {
		for _, pkg := range strings.Split(string(out), "\n") {
			if !strings.HasPrefix(pkg, "./client") && !strings.HasPrefix(pkg, "./models") {
				pp = append(pp, pkg)
			}
		}
	}
	return append(pp, ".")
}

func containsGoSources(dir string) bool {
	out, err := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go").CombinedOutput()
	return err == nil && len(out) > 0
}
