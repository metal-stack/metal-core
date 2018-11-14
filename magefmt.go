//+build mage

package main

import (
	"os/exec"
	"strings"
)

// Format source code
func Fmt() {
	for _, pkg := range fetchGoPackages() {
		if containsGoSources(pkg) {
			exec.Command("go", "fmt", pkg).Run()
		}
	}
}

func fetchGoPackages() []string {
	if out, err := exec.Command("find", "./cmd", "-mindepth", "1", "-type", "d").CombinedOutput(); err == nil && len(out) > 0 {
		return append(append(strings.Split(string(out), "\n"), "."), "./domain")
	}
	return []string{}
}

func containsGoSources(dir string) bool {
	out, err := exec.Command("find", dir, "-maxdepth", "1", "-type", "f", "-name", "*.go").CombinedOutput()
	return err == nil && len(out) > 0
}
