package main

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/build"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metalcore"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spec" {
		build.Spec()
		return
	}

	server := metalcore.Create()
	server.Run()
}
