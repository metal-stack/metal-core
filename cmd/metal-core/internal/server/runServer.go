package server

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"go.uber.org/zap"
	"net/http"
)

func (s server) Run() {
	restful.DefaultContainer.Add(s.Endpoint().NewBootService())
	restful.DefaultContainer.Add(s.Endpoint().NewDeviceService())

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(),
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	// enable CORS for the UI to work
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer,
	}
	restful.DefaultContainer.Filter(cors.Filter)

	addr := fmt.Sprintf("%v:%d", s.Config.BindAddress, s.Config.Port)

	zapup.MustRootLogger().Info("Starting metal-core",
		zap.String("address", addr),
	)

	http.ListenAndServe(addr, nil)
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
