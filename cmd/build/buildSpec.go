package build

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/endpoint"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

func Spec() {
	cfg := core.Init(endpoint.NewHandler(nil))
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", js)
}
