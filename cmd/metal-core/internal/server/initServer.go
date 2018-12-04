package server

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
)

func Init(e domain.Endpoint) *restfulspec.Config {
	restful.DefaultContainer.Add(e.NewBootService())
	restful.DefaultContainer.Add(e.NewDeviceService())

	cfg := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(),
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(cfg))
	return &cfg
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "metal-core",
			Description: "Resource for managing PXE clients",
			Contact: &spec.ContactInfo{
				Name:  "Devops Team",
				Email: "devops@f-i-ts.de",
				URL:   "http://www.f-i-ts.de",
			},
			License: &spec.License{
				Name: "MIT",
				URL:  "http://mit.org",
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{
		spec.Tag{TagProps: spec.TagProps{
			Name:        "boot",
			Description: "Booting PXE clients"}},
		spec.Tag{TagProps: spec.TagProps{
			Name:        "device",
			Description: "Managing PXE boot clients"},
		},
	}
}
