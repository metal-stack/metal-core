//+build mage

package main

import (
	"github.com/magefile/mage/sh"
	"os/exec"
	"strings"
)

// Format all metal-core sources
func Fmt() {
	for _, pkg := range fetchGoPackages() {
		if containsGoSources(pkg) {
			sh.RunV("go", "fmt", pkg)
		}
	}
}

func fetchGoPackages() []string {
	if out, err := exec.Command("find", ".", "-mindepth", "1", "-type", "d").CombinedOutput(); err == nil && len(out) > 0 {
		return append(strings.Split(string(out), "\n"), ".")
	} else {
		return []string{}
	}
}

func containsGoSources(dir string) bool {
	out, err := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go").CombinedOutput()
	return err == nil && len(out) > 0
}
