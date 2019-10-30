package main

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/build"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metalcore"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "spec" {
		filename := ""
		if len(os.Args) > 2 {
			filename = os.Args[2]
		}
		build.Spec(filename)
		return
	}

	server := metalcore.Create()
	server.Run()
}
