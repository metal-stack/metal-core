//+build mage

package main

import (
	"os"
	"os/exec"
)

// Format source code
func Fmt() {
	os.Setenv("GO111MODULE", "off")
	defer os.Setenv("GO111MODULE", "on")
	exec.Command("go", "fmt", "./...")
}
