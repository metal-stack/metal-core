//+build mage

package main

import (
	"github.com/magefile/mage/sh"
	"os/exec"
	"strings"
)

// Format all metalcore sources
func Fmt() {
	sh.Run("go", "fmt", ".")
	findPackages := exec.Command("find", ".", "-mindepth", "1", "-type", "d", "-not", "-regex", ".*/\\..*", "-and", "-not", "-regex", ".*/bin.*")
	if out, err := findPackages.CombinedOutput(); err == nil {
		packages := strings.Split(string(out[:len(out)-1]), "\n")
		for _, pkg := range packages {
			if containsGoSources(pkg) {
				sh.Run("go", "fmt", pkg)
			}
		}
	}
}

func containsGoSources(dir string) bool {
	findGoSources := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go")
	out, err := findGoSources.CombinedOutput()
	return err == nil && len(out) > 0
}
