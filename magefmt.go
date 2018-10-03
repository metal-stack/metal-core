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
			sh.Run("go", "fmt", pkg)
		}
	}
}

func fetchGoPackages() []string {
	cmd := exec.Command("find", ".", "-mindepth", "1", "-type", "d")
	if out, err := cmd.CombinedOutput(); err == nil && len(out) > 0 {
		return append(strings.Split(string(out[:len(out)-1]), "\n"), ".")
	} else {
		return []string{}
	}
}

func containsGoSources(dir string) bool {
	cmd := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go")
	out, err := cmd.CombinedOutput()
	return err == nil && len(out) > 0
}
