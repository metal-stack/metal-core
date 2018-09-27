package metal_api

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
)

var Config domain.Config

func get(path string, query map[string]string, domainObject interface{}) int {
	return rest.Get(Config.MetalApiProtocol, Config.MetalApiAddress, Config.MetalApiPort, path, query, &domainObject)
}

func FindDevice(mac string) (int, []*domain.Device) {
	device := []*domain.Device{}
	statusCode := get("/device/find", rest.CreateQueryParameters("mac", mac), device)
	return statusCode, device
}