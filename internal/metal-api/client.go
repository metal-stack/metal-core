package metal_api

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
)

var Config domain.Config

func FindDevices(mac string) (int, []domain.Device) {
	var devices []domain.Device
	statusCode := get("/device/find", rest.CreateQueryParameters("mac", mac), &devices)
	return statusCode, devices
}

func RegisterDevice(lshw string) (int, domain.Device) {
	request := domain.RegisterDeviceRequest{}
	//TODO populate request with appropriate values from lshw
	var device domain.Device
	statusCode := post("/device/register", request, &device)
	return statusCode, device
}

func get(path string, query map[string]string, domainObject interface{}) int {
	return rest.Get(Config.MetalApiProtocol, Config.MetalApiAddress, Config.MetalApiPort, path, query, domainObject)
}

func post(path string, body interface{}, domainObject interface{}) int {
	return rest.Post(Config.MetalApiProtocol, Config.MetalApiAddress, Config.MetalApiPort, path, body, domainObject)
}
