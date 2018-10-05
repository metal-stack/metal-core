package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
)

func (c client) FindDevices(mac string) (int, []domain.Device) {
	var devs []domain.Device
	sc := c.getExpect("/device/find", rest.CreateQueryParams("mac", mac), &devs)
	return sc, devs
}
