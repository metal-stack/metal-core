package build

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/endpoint"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"io/ioutil"
)

func Spec(filename string) {
	cfg := core.Init(endpoint.NewHandler(nil))
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, js, 0644); err != nil {
		fmt.Printf("%s\n", js)
	}
}
