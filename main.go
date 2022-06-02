package main

import (
	"github.com/metal-stack/metal-core/cmd/build"
	"github.com/metal-stack/metal-core/cmd/metalcore"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spec" {
		build.Spec()
		return
	}

	app := metalcore.Create()
	app.Run()
}
