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
		Config() *domain.Config
		FindDevices(mac string) (int, []*models.MetalDevice)
		RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice)
		InstallImage(deviceId string) (int, *models.MetalDevice)
	}
	client struct {
		device *device.Client
		config *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	transport := httptransport.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)
	return client{
		device: device.New(transport, strfmt.Default),
		config: cfg,
	}
}

func (c client) Device() *device.Client {
	return c.device
}

func (c client) Config() *domain.Config {
	return c.config
}
