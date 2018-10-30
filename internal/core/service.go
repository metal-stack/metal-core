package core

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/netswitch"
	log "github.com/sirupsen/logrus"
)

type (
	Service interface {
		GetConfig() *domain.Config
		GetMetalAPIClient() api.Client
		GetNetSwitchClient() netswitch.Client
		GetServer() *http.Server
		RunServer()
	}
	service struct {
		server          *http.Server
		apiClient       api.Client
		netSwitchClient netswitch.Client
	}
)

var srv Service

func NewService(cfg *domain.Config) Service {
	srv = service{
		server:          &http.Server{},
		apiClient:       api.NewClient(cfg),
		netSwitchClient: netswitch.NewClient(cfg),
	}
	return srv
}

func (s service) GetConfig() *domain.Config {
	return s.GetMetalAPIClient().GetConfig()
}

func (s service) GetMetalAPIClient() api.Client {
	return s.apiClient
}

func (s service) GetNetSwitchClient() netswitch.Client {
	return s.netSwitchClient
}

func (s service) GetServer() *http.Server {
	return s.server
}

func (s service) RunServer() {
	restful.DefaultContainer.Add(NewBootService())
	restful.DefaultContainer.Add(NewDeviceService())

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(),
		APIPath:                       "/apidocs.yaml",
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

	addr := fmt.Sprintf("%v:%d", s.GetConfig().Address, s.GetConfig().Port)

	log.WithFields(log.Fields{
		"address": addr,
	}).Info("Starting metal-core")

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
