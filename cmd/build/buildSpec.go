package build

import (
	"encoding/json"
	"fmt"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/metal-stack/metal-core/internal/core"
	"github.com/metal-stack/metal-core/internal/endpoint"
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
