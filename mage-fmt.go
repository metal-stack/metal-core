//+build mage

package main

import (
	"github.com/magefile/mage/sh"
	"os/exec"
	"strings"
)

// Format all metalcore sources
func Fmt() {
	for _, pkg := range fetchGoPackages() {
		if containsGoSources(pkg) {
			sh.Run("go", "fmt", pkg)
		}
	}
}

func fetchGoPackages() []string {
	findPackages := exec.Command("find", ".", "-mindepth", "1", "-type", "d")
	if out, err := findPackages.CombinedOutput(); err == nil && len(out) > 0 {
		return append(strings.Split(string(out[:len(out)-1]), "\n"), ".")
	} else {
		return []string{}
	}
}

func containsGoSources(dir string) bool {
	findGoSources := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go")
	out, err := findGoSources.CombinedOutput()
	return err == nil && len(out) > 0
}
