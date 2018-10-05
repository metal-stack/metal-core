//+build mage

package main

import (
	"github.com/magefile/mage/sh"
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
	if out, err := sh.Output("find", ".", "-mindepth", "1", "-type", "d"); err == nil && len(out) > 0 {
		return append(strings.Split(out, "\n"), ".")
	} else {
		return []string{}
	}
}

func containsGoSources(dir string) bool {
	out, err := sh.Output("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go")
	return err == nil && len(out) > 0
}
