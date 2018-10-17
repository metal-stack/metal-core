package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

type (
	Client interface {
		Device() *device.Client
		GetConfig() *domain.Config
		FindDevice(mac string) (int, *models.MetalDevice)
		RegisterDevice(deviceId string, hw []byte) (int, *models.MetalDevice)
		InstallImage(deviceId string) (int, *models.MetalDevice)
	}
	client struct {
		DeviceClient *device.Client
		Config       *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	transport := httptransport.New(fmt.Sprintf("%v:%d", cfg.APIAddress, cfg.APIPort), "", nil)
	return client{
		DeviceClient: device.New(transport, strfmt.Default),
		Config:       cfg,
	}
}

func (c client) Device() *device.Client {
	return c.Device()
}

func (c client) GetConfig() *domain.Config {
	return c.Config
}
