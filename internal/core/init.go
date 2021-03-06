package core

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/go-openapi/spec"
	"github.com/metal-stack/metal-core/pkg/domain"
)

func Init(endpointHandler domain.EndpointHandler) *restfulspec.Config {
	restful.DefaultContainer.Add(endpointHandler.NewBootService())
	restful.DefaultContainer.Add(endpointHandler.NewMachineService())
	restful.DefaultContainer.Add(endpointHandler.NewCertsService())

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
				ContactInfoProps: spec.ContactInfoProps{
					Name:  "Devops Team",
					Email: "devops@f-i-ts.de",
					URL:   "http://www.f-i-ts.de",
				},
			},
			License: &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: "MIT",
					URL:  "http://mit.org",
				},
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{
		{TagProps: spec.TagProps{
			Name:        "boot",
			Description: "Booting PXE clients"}},
		{TagProps: spec.TagProps{
			Name:        "machine",
			Description: "Managing PXE boot clients"},
		},
	}
}
