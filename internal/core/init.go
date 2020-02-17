package core

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
)

func Init(endpointHandler domain.EndpointHandler) *restfulspec.Config {
	restful.DefaultContainer.Add(endpointHandler.NewBootService())
	restful.DefaultContainer.Add(endpointHandler.NewMachineService())

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
