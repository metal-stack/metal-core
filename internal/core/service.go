package core

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/log"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"go.uber.org/zap"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
)

type (
	Service interface {
		Config() *domain.Config
		API() api.Client
		IPMI() *ipmi.IpmiConnection
		Server() *http.Server
		FreeDevice(device *models.MetalDevice)
		RunServer()
	}
	service struct {
		server    *http.Server
		apiClient api.Client
		ipmiConn  *ipmi.IpmiConnection
	}
)

var (
	srv Service
)

func NewService(cfg *domain.Config) Service {
	srv = service{
		server:    &http.Server{},
		apiClient: api.NewClient(cfg),
		ipmiConn: &ipmi.IpmiConnection{
			// Requires gateway of the control plane for running in Metal Lab... this is just a quick workaround for the poc
			Hostname:  cfg.IP[:strings.LastIndex(cfg.IP, ".")] + ".1",
			Interface: "lanplus",
			Port:      6230,
			Username:  "vagrant",
			Password:  "vagrant",
		},
	}
	return srv
}

func (s service) Config() *domain.Config {
	return s.API().Config()
}

func (s service) API() api.Client {
	return s.apiClient
}

func (s service) Server() *http.Server {
	return s.server
}

func (s service) IPMI() *ipmi.IpmiConnection {
	return s.ipmiConn
}

func (s service) FreeDevice(device *models.MetalDevice) {
	if err := ipmi.SetBootDevPxe(srv.IPMI()); err != nil {
		log.Get().Error("Unable to set boot order of device to HD",
			zap.Any("device", device),
			zap.Error(err),
		)
	} else {
		log.Get().Info("Freed device",
			zap.Any("device", device),
		)
	}
}

func (s service) RunServer() {
	restful.DefaultContainer.Add(NewBootService())
	restful.DefaultContainer.Add(NewDeviceService())

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

	addr := fmt.Sprintf("%v:%d", s.Config().BindAddress, s.Config().Port)

	log.Get().Info("Starting metal-core",
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
